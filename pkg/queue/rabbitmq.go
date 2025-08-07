package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
	"creative-studio-server/config"
	"creative-studio-server/pkg/logger"
)

type RabbitMQClient struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queues     map[string]amqp.Queue
}

type Task struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Priority  int                    `json:"priority"`
	Retry     int                    `json:"retry"`
	MaxRetry  int                    `json:"max_retry"`
	CreatedAt time.Time              `json:"created_at"`
}

type TaskHandler func(task *Task) error

var Queue *RabbitMQClient

func InitRabbitMQ(cfg *config.Config) error {
	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	Queue = &RabbitMQClient{
		connection: conn,
		channel:    ch,
		queues:     make(map[string]amqp.Queue),
	}

	// Declare default queues
	if err := Queue.declareQueues(); err != nil {
		return fmt.Errorf("failed to declare queues: %w", err)
	}

	logger.Info("RabbitMQ connected successfully")
	return nil
}

func (r *RabbitMQClient) declareQueues() error {
	queueNames := []string{
		"video_processing",
		"smart_composition",
		"render_tasks",
		"analysis_tasks",
		"thumbnail_generation",
	}

	for _, name := range queueNames {
		queue, err := r.channel.QueueDeclare(
			name,
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			amqp.Table{
				"x-message-ttl":                 int32(30 * 60 * 1000), // 30 minutes
				"x-dead-letter-exchange":        "dlx",
				"x-dead-letter-routing-key":     "dlx." + name,
				"x-max-priority":                int32(10),
			},
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", name, err)
		}

		r.queues[name] = queue
	}

	// Declare dead letter exchange
	err := r.channel.ExchangeDeclare(
		"dlx",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %w", err)
	}

	return nil
}

func (r *RabbitMQClient) PublishTask(queueName string, task *Task) error {
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	priority := uint8(task.Priority)
	if priority > 10 {
		priority = 10
	}

	err = r.channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Priority:     priority,
			Timestamp:    time.Now(),
			DeliveryMode: amqp.Persistent,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish task to queue %s: %w", queueName, err)
	}

	logger.Infof("Task published to queue %s: %s", queueName, task.ID)
	return nil
}

func (r *RabbitMQClient) ConsumeTask(queueName string, handler TaskHandler, concurrency int) error {
	// Set QoS for the channel
	err := r.channel.Qos(
		concurrency, // prefetch count
		0,           // prefetch size
		false,       // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := r.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	// Start consumer goroutines
	for i := 0; i < concurrency; i++ {
		go r.worker(msgs, handler, queueName)
	}

	logger.Infof("Started %d workers for queue %s", concurrency, queueName)
	return nil
}

func (r *RabbitMQClient) worker(msgs <-chan amqp.Delivery, handler TaskHandler, queueName string) {
	for msg := range msgs {
		var task Task
		if err := json.Unmarshal(msg.Body, &task); err != nil {
			logger.Errorf("Failed to unmarshal task from queue %s: %v", queueName, err)
			msg.Nack(false, false) // Dead letter
			continue
		}

		logger.Infof("Processing task %s from queue %s", task.ID, queueName)

		err := handler(&task)
		if err != nil {
			logger.Errorf("Task %s failed: %v", task.ID, err)

			// Retry logic
			if task.Retry < task.MaxRetry {
				task.Retry++
				if retryErr := r.PublishTask(queueName, &task); retryErr != nil {
					logger.Errorf("Failed to retry task %s: %v", task.ID, retryErr)
				} else {
					logger.Infof("Task %s queued for retry (%d/%d)", task.ID, task.Retry, task.MaxRetry)
				}
			}

			msg.Nack(false, false) // Dead letter after max retries
		} else {
			logger.Infof("Task %s completed successfully", task.ID)
			msg.Ack(false)
		}
	}
}

func (r *RabbitMQClient) CreateTask(taskType string, payload map[string]interface{}, priority int) *Task {
	return &Task{
		ID:        generateTaskID(),
		Type:      taskType,
		Payload:   payload,
		Priority:  priority,
		Retry:     0,
		MaxRetry:  3,
		CreatedAt: time.Now(),
	}
}

func (r *RabbitMQClient) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.connection != nil {
		return r.connection.Close()
	}
	return nil
}

// Task type constants
const (
	TaskTypeVideoProcessing      = "video_processing"
	TaskTypeSmartComposition     = "smart_composition"
	TaskTypeRenderVideo          = "render_video"
	TaskTypeAnalyzeVideo         = "analyze_video"
	TaskTypeGenerateThumbnail    = "generate_thumbnail"
	TaskTypeExtractAudio         = "extract_audio"
	TaskTypeApplyEffects         = "apply_effects"
)

// Helper functions for different task types
func PublishVideoProcessingTask(clipID uint, filePath string) error {
	task := Queue.CreateTask(TaskTypeVideoProcessing, map[string]interface{}{
		"clip_id":   clipID,
		"file_path": filePath,
	}, 5)

	return Queue.PublishTask("video_processing", task)
}

func PublishSmartCompositionTask(projectID uint, requirements map[string]interface{}) error {
	task := Queue.CreateTask(TaskTypeSmartComposition, map[string]interface{}{
		"project_id":    projectID,
		"requirements":  requirements,
	}, 7)

	return Queue.PublishTask("smart_composition", task)
}

func PublishRenderTask(taskID string, renderOptions map[string]interface{}) error {
	task := Queue.CreateTask(TaskTypeRenderVideo, map[string]interface{}{
		"task_id":        taskID,
		"render_options": renderOptions,
	}, 8)

	return Queue.PublishTask("render_tasks", task)
}

func PublishAnalysisTask(clipID uint, analysisType string) error {
	task := Queue.CreateTask(TaskTypeAnalyzeVideo, map[string]interface{}{
		"clip_id":       clipID,
		"analysis_type": analysisType,
	}, 3)

	return Queue.PublishTask("analysis_tasks", task)
}

func PublishThumbnailTask(clipID uint, filePath string) error {
	task := Queue.CreateTask(TaskTypeGenerateThumbnail, map[string]interface{}{
		"clip_id":   clipID,
		"file_path": filePath,
	}, 2)

	return Queue.PublishTask("thumbnail_generation", task)
}

func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

// Task Handlers

func VideoProcessingHandler(task *Task) error {
	clipID, ok := task.Payload["clip_id"].(float64) // JSON numbers are float64
	if !ok {
		return fmt.Errorf("invalid clip_id in task payload")
	}

	filePath, ok := task.Payload["file_path"].(string)
	if !ok {
		return fmt.Errorf("invalid file_path in task payload")
	}

	logger.Infof("Processing video for clip %d: %s", uint(clipID), filePath)
	
	// TODO: Implement actual video processing logic
	// This would include:
	// - Video analysis
	// - Thumbnail generation
	// - Metadata extraction
	// - Quality assessment

	time.Sleep(2 * time.Second) // Simulate processing time
	
	return nil
}

func SmartCompositionHandler(task *Task) error {
	projectID, ok := task.Payload["project_id"].(float64)
	if !ok {
		return fmt.Errorf("invalid project_id in task payload")
	}

	logger.Infof("Generating smart composition for project %d", uint(projectID))
	
	// TODO: Implement smart composition logic
	// This would include:
	// - Fetching clips
	// - Running composition algorithms
	// - Generating timeline
	// - Storing results

	time.Sleep(5 * time.Second) // Simulate processing time
	
	return nil
}

func RenderTaskHandler(task *Task) error {
	taskID, ok := task.Payload["task_id"].(string)
	if !ok {
		return fmt.Errorf("invalid task_id in task payload")
	}

	logger.Infof("Rendering video for task %s", taskID)
	
	// TODO: Implement video rendering logic
	// This would include:
	// - Fetching render parameters
	// - Running FFmpeg commands
	// - Progress tracking
	// - Result storage

	time.Sleep(10 * time.Second) // Simulate rendering time
	
	return nil
}

func AnalysisTaskHandler(task *Task) error {
	clipID, ok := task.Payload["clip_id"].(float64)
	if !ok {
		return fmt.Errorf("invalid clip_id in task payload")
	}

	analysisType, ok := task.Payload["analysis_type"].(string)
	if !ok {
		return fmt.Errorf("invalid analysis_type in task payload")
	}

	logger.Infof("Analyzing clip %d with type %s", uint(clipID), analysisType)
	
	// TODO: Implement video analysis logic
	// This would include:
	// - Content analysis
	// - AI-based tagging
	// - Motion detection
	// - Color analysis

	time.Sleep(3 * time.Second) // Simulate analysis time
	
	return nil
}

func ThumbnailTaskHandler(task *Task) error {
	clipID, ok := task.Payload["clip_id"].(float64)
	if !ok {
		return fmt.Errorf("invalid clip_id in task payload")
	}

	filePath, ok := task.Payload["file_path"].(string)
	if !ok {
		return fmt.Errorf("invalid file_path in task payload")
	}

	logger.Infof("Generating thumbnail for clip %d: %s", uint(clipID), filePath)
	
	// TODO: Implement thumbnail generation logic
	// This would use FFmpeg to extract frames

	time.Sleep(1 * time.Second) // Simulate thumbnail generation time
	
	return nil
}