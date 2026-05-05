package rag

import (
	"Checkin/internal/dao"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type RAGService struct {
	client         *openai.Client
	embeddingModel openai.EmbeddingModel
	dimension      int
	topK           int
	uploadDir      string
}

func NewRAGService(cfg *viper.Viper) *RAGService {
	apiKey := cfg.GetString("llm.api_key")
	baseURL := cfg.GetString("llm.base_url")

	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	return &RAGService{
		client:         openai.NewClientWithConfig(config),
		embeddingModel: openai.EmbeddingModel(cfg.GetString("rag.embedding_model")),
		dimension:      cfg.GetInt("rag.dimension"),
		topK:           cfg.GetInt("rag.top_k"),
		uploadDir:      cfg.GetString("rag.upload_dir"),
	}
}

// IndexFile 读取文件内容，切块后向量化并存入 Redis
func (s *RAGService) IndexFile(ctx context.Context, userID int64, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	filename := filepath.Base(filePath)
	chunks := splitText(string(content), 500)
	if len(chunks) == 0 {
		return fmt.Errorf("文件内容为空")
	}

	// 先清除该用户的旧索引
	if err := s.deleteUserIndex(ctx, userID); err != nil {
		zap.L().Warn("清除旧索引失败", zap.Error(err))
	}

	// 确保 Redis 向量索引存在
	if err := s.ensureIndex(ctx, userID); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	// 批量生成向量
	embeddings, err := s.getEmbeddings(ctx, chunks)
	if err != nil {
		return fmt.Errorf("向量化失败: %w", err)
	}

	// 写入 Redis
	rdb := dao.RedisClient
	prefix := indexKeyPrefix(userID)
	pipe := rdb.Pipeline()

	for i, chunk := range chunks {
		key := fmt.Sprintf("%sdoc:%d", prefix, i)
		vectorBytes := float32SliceToBytes(embeddings[i])

		pipe.HSet(ctx, key, map[string]interface{}{
			"content":  chunk,
			"source":   filename,
			"chunk_id": i,
			"vector":   vectorBytes,
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("写入 Redis 失败: %w", err)
	}

	zap.L().Info("RAG 索引创建成功",
		zap.Int64("userID", userID),
		zap.String("file", filename),
		zap.Int("chunks", len(chunks)),
	)
	return nil
}

// Retrieve 根据查询文本检索最相关的文档块
func (s *RAGService) Retrieve(ctx context.Context, userID int64, query string) ([]string, error) {
	// 将查询文本向量化
	embeddings, err := s.getEmbeddings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("查询向量化失败: %w", err)
	}
	queryVector := float32SliceToBytes(embeddings[0])

	rdb := dao.RedisClient
	indexName := indexName(userID)

	// Redis FT.SEARCH KNN 查询
	searchCmd := rdb.Do(ctx,
		"FT.SEARCH", indexName,
		fmt.Sprintf("*=>[KNN %d @vector $BLOB AS distance]", s.topK),
		"PARAMS", "2", "BLOB", string(queryVector),
		"SORTBY", "distance",
		"RETURN", "2", "content", "distance",
		"DIALECT", "2",
	)

	result, err := searchCmd.Result()
	if err != nil {
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	return parseSearchResult(result), nil
}

// BuildRAGPrompt 构建包含检索文档的提示词
func BuildRAGPrompt(query string, docs []string) string {
	if len(docs) == 0 {
		return query
	}

	var sb strings.Builder
	sb.WriteString("基于以下参考文档回答用户的问题。如果文档中没有相关信息，请说明无法从知识库中找到相关信息。\n\n参考文档：\n")
	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("[文档 %d]: %s\n\n", i+1, doc))
	}
	sb.WriteString(fmt.Sprintf("用户问题：%s\n\n请提供准确、完整的回答：", query))
	return sb.String()
}

// GetUserUploadDir 获取用户的上传目录
func (s *RAGService) GetUserUploadDir(userID int64) string {
	return filepath.Join(s.uploadDir, fmt.Sprintf("%d", userID))
}

// HasUserIndex 检查用户是否有 RAG 索引
func (s *RAGService) HasUserIndex(ctx context.Context, userID int64) bool {
	rdb := dao.RedisClient
	name := indexName(userID)
	err := rdb.Do(ctx, "FT.INFO", name).Err()
	return err == nil
}

// --- 内部方法 ---

func (s *RAGService) getEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	resp, err := s.client.CreateEmbeddings(ctx, openai.EmbeddingRequestStrings{
		Input:          texts,
		Model:          s.embeddingModel,
		EncodingFormat: openai.EmbeddingEncodingFormatFloat,
	})
	if err != nil {
		return nil, err
	}

	result := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		result[i] = d.Embedding
	}
	return result, nil
}

func (s *RAGService) ensureIndex(ctx context.Context, userID int64) error {
	rdb := dao.RedisClient
	name := indexName(userID)

	// 检查索引是否已存在
	err := rdb.Do(ctx, "FT.INFO", name).Err()
	if err == nil {
		return nil
	}

	prefix := indexKeyPrefix(userID)
	return rdb.Do(ctx,
		"FT.CREATE", name,
		"ON", "HASH",
		"PREFIX", "1", prefix,
		"SCHEMA",
		"content", "TEXT",
		"source", "TEXT",
		"chunk_id", "NUMERIC",
		"vector", "VECTOR", "FLAT",
		"6",
		"TYPE", "FLOAT32",
		"DIM", s.dimension,
		"DISTANCE_METRIC", "COSINE",
	).Err()
}

func (s *RAGService) deleteUserIndex(ctx context.Context, userID int64) error {
	rdb := dao.RedisClient
	name := indexName(userID)

	// 删除索引（DD = 同时删除关联的 Hash 数据）
	err := rdb.Do(ctx, "FT.DROPINDEX", name, "DD").Err()
	if err != nil && !strings.Contains(err.Error(), "Unknown") {
		return err
	}
	return nil
}

// --- 工具函数 ---

func indexName(userID int64) string {
	return fmt.Sprintf("rag:idx:user:%d", userID)
}

func indexKeyPrefix(userID int64) string {
	return fmt.Sprintf("rag:user:%d:", userID)
}

// splitText 按最大长度切块，尽量在换行符处断开
func splitText(text string, maxLen int) []string {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return nil
	}
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	paragraphs := strings.Split(text, "\n")
	var current strings.Builder

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if current.Len()+len(p)+1 > maxLen && current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(p)
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

func float32SliceToBytes(floats []float32) []byte {
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func parseSearchResult(result interface{}) []string {
	arr, ok := result.([]interface{})
	if !ok || len(arr) < 2 {
		return nil
	}

	var docs []string
	// arr[0] 是总数，之后每2个元素为 [key, fields...]
	for i := 1; i < len(arr); i += 2 {
		if i+1 >= len(arr) {
			break
		}
		fields, ok := arr[i+1].([]interface{})
		if !ok {
			continue
		}
		for j := 0; j < len(fields)-1; j += 2 {
			key, _ := fields[j].(string)
			if key == "content" {
				val, _ := fields[j+1].(string)
				if val != "" {
					docs = append(docs, val)
				}
			}
		}
	}
	return docs
}

// DeleteUserFiles 删除用户上传目录下的所有文件
func (s *RAGService) DeleteUserFiles(userID int64) error {
	dir := s.GetUserUploadDir(userID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
	return nil
}

// ValidateFile 校验文件类型
func ValidateFile(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".md" && ext != ".txt" {
		return fmt.Errorf("只允许上传 .md 或 .txt 文件，当前: %s", ext)
	}
	return nil
}

// UploadAndIndex 完整的上传+索引流程
func (s *RAGService) UploadAndIndex(ctx context.Context, userID int64, filename string, content []byte) (string, error) {
	if err := ValidateFile(filename); err != nil {
		return "", err
	}

	userDir := s.GetUserUploadDir(userID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 清理旧文件
	s.DeleteUserFiles(userID)

	filePath := filepath.Join(userDir, filename)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	if err := s.IndexFile(ctx, userID, filePath); err != nil {
		os.Remove(filePath)
		return "", fmt.Errorf("索引失败: %w", err)
	}

	return filePath, nil
}

// ensure RedisClient type is properly used
var _ *redis.Client = dao.RedisClient
