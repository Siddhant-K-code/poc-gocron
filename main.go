package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-co-op/gocron/v2"
	"github.com/kelseyhightower/envconfig"
	nid "github.com/matoous/go-nanoid/v2"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gopkg.in/yaml.v3"
)

// Config represents the overall configuration needed for the backup tool
type Config struct {
	StorageConfig StorageDetails `envconfig:"STORAGE"`
	PathToConfig  string         `envconfig:"CONFIG_PATH" required:"true"`
}

// StorageDetails encapsulates the details necessary for S3 storage access
type StorageDetails struct {
	ServerURL       string `envconfig:"S3_ENDPOINT" required:"true"`
	Location        string `envconfig:"S3_REGION" required:"true"`
	Container       string `envconfig:"S3_BUCKET" required:"true"`
	PrivateKey      string `envconfig:"S3_SECRET_KEY" required:"true"`
	PublicKey       string `envconfig:"S3_ACCESS_KEY" required:"true"`
	CreateIfMissing bool   `envconfig:"S3_AUTO_CREATE_BUCKET" default:"false"`
}

// BackupSpecifications defines how backup tasks are structured
type BackupSpecifications struct {
	Tasks []BackupTask `yaml:"jobs"`
}

func main() {
	var settings Config
	if err := envconfig.Process("", &settings); err != nil {
		slog.Error("Failed to load environment variables", slog.String("error", err.Error()))
		return
	}

	minioClient, err := minio.New(settings.StorageConfig.ServerURL, &minio.Options{
		Creds:  credentials.NewStaticV4(settings.StorageConfig.PublicKey, settings.StorageConfig.PrivateKey, ""),
		Secure: true,
	})
	if err != nil {
		slog.Error("Failed to initialize MinIO client", slog.String("error", err.Error()))
		return
	}

	bucketExists, err := minioClient.BucketExists(context.Background(), settings.StorageConfig.Container)
	if err != nil {
		slog.Error("Failed to check if bucket exists", slog.String("error", err.Error()))
		return
	}

	if !bucketExists {
		if settings.StorageConfig.CreateIfMissing {
			if err := minioClient.MakeBucket(context.Background(), settings.StorageConfig.Container, minio.MakeBucketOptions{Region: settings.StorageConfig.Location}); err != nil {
				slog.Error("Failed to create bucket", slog.String("error", err.Error()))
				return
			}
			slog.Info("Bucket was successfully created", slog.String("bucket", settings.StorageConfig.Container))
		} else {
			slog.Error("Bucket does not exist", slog.String("bucket", settings.StorageConfig.Container))
			return
		}
	}

	var backupPlans BackupSpecifications
	if err := loadBackupConfig(settings.PathToConfig, &backupPlans); err != nil {
		slog.Error("Failed to load backup configuration", slog.String("error", err.Error()))
		return
	}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		fmt.Printf("Failed to create a scheduler: %s\n", err)
		return
	}

	scheduler.Start()

	for _, task := range backupPlans.Tasks {
		if _, err := scheduler.NewJob(
			gocron.CronJob(task.Schedule, false),
			gocron.NewTask(task.Execute(minioClient, settings.StorageConfig.Container)),
		); err != nil {
			slog.Error("Failed to schedule backup job", slog.String("error", err.Error()), slog.String("backup_task", task.Name))
			return
		}
	}

	slog.Info("Scheduler has started")
	waitForTermination()
	slog.Info("Scheduler is stopping")
}

func loadBackupConfig(path string, specs *BackupSpecifications) error {
	fileData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %s", err)
	}
	if err := yaml.Unmarshal(fileData, specs); err != nil {
		return fmt.Errorf("failed to parse configuration file: %s", err)
	}
	return nil
}

type BackupTask struct {
	Name           string   `yaml:"name"`
	Schedule       string   `yaml:"schedule"`
	Commands       []string `yaml:"script"`
	TargetFilePath string   `yaml:"filepath_to_upload"`
}

func (task BackupTask) Execute(client *minio.Client, bucketName string) func() {
	slog.Info("Preparing to execute backup task", slog.String("backup_task", task.Name))

	return func() {
		backupID, _ := nid.Generate("1234567890abcdefghijklmnopqrstuvwxyz", 8)
		logger := slog.With(
			slog.String("id", backupID),
			slog.String("backup_task", task.Name),
		)

		logger.Info("Backup task started")
		defer logger.Info("Backup task completed")

		tempDir, err := createTemporaryDirectory(task.Name, backupID)
		if err != nil {
			logger.Error("Failed to create a temporary directory", slog.String("error", err.Error()))
			return
		}

		processScripts(task.Commands, tempDir, backupID)
		if err := executeBackup(task.Commands, logger); err != nil {
			logger.Error("Failed during backup execution", slog.String("error", err.Error()))
			return
		}

		if _, err := validateFile(task.TargetFilePath); err != nil {
			logger.Error("Failed to validate the backup file", slog.String("error", err.Error()))
			return
		}

		fileExtension := filepath.Ext(task.TargetFilePath)
		newFileName := generateFileName(task.Name, backupID, fileExtension)
		if mimeType, err := detectMimeType(task.TargetFilePath); err != nil {
			logger.Error("Failed to detect MIME type of the file", slog.String("error", err.Error()))
			return
		} else {
			uploadFile(client, bucketName, newFileName, task.TargetFilePath, mimeType, logger)
		}
	}
}

func createTemporaryDirectory(name, id string) (string, error) {
	directoryPath := fmt.Sprintf("%s/backup-%s-%s-", os.TempDir(), name, id)
	return os.MkdirTemp(directoryPath, "")
}

func processScripts(scripts []string, tempDir, id string) {
	for i, script := range scripts {
		scripts[i] = replaceTemplate(script, id, tempDir)
	}
}

func executeBackup(scripts []string, logger *slog.Logger) error {
	cmd := exec.Command("sh", "-c", strings.Join(scripts, " \n"))
	cmd.Stderr = newLogger(logger, true)
	cmd.Stdout = newLogger(logger, false)
	return cmd.Run()
}

func validateFile(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		return false, err
	}
	return true, nil
}

func generateFileName(baseName, id, extension string) string {
	timestamp := time.Now().Format("2006_01_02_02_15_04_05")
	return fmt.Sprintf("%s-%s-%s%s", timestamp, baseName, id, extension)
}

func detectMimeType(filePath string) (string, error) {
	mtype, err := mimetype.DetectFile(filePath)
	if err != nil {
		return "", err
	}
	return mtype.String(), nil
}

func uploadFile(client *minio.Client, bucket, fileName, filePath, mimeType string, logger *slog.Logger) {
	if _, err := client.FPutObject(
		context.Background(),
		bucket,
		fileName,
		filePath,
		minio.PutObjectOptions{
			ContentType: mimeType,
		},
	); err != nil {
		logger.Error("Failed to upload the file to object storage", slog.String("error", err.Error()))
	}
}

func replaceTemplate(original, id, tempDir string) string {
	replacements := map[string]string{
		"${BACKUP_ID}":   id,
		"${TEMP_DIR}":    tempDir,
		"${BACKUP_NAME}": original,
	}
	for key, val := range replacements {
		original = strings.ReplaceAll(original, key, val)
	}
	return original
}

func newLogger(logger *slog.Logger, isError bool) *CommandLogger {
	return &CommandLogger{l: logger, err: isError}
}

type CommandLogger struct {
	l   *slog.Logger
	err bool
}

func (c *CommandLogger) Write(data []byte) (int, error) {
	message := strings.TrimRight(string(data), "\n")
	message = strings.ReplaceAll(message, "\n", "\\n")
	if c.err {
		c.l.Error("SCRIPT> " + message)
	} else {
		c.l.Info("SCRIPT> " + message)
	}
	return len(data), nil
}

func waitForTermination() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)
	<-signals
}
