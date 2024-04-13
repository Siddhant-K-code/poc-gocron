# 🚀 poc-gocron: A Simple S3 Backup Tool 🚀

## 🌟 Overview

It's a sleek and simple backup tool crafted in Go, designed as a proof of concept (POC) for [`gocron`](https://github.com/go-co-op/gocron). 🛠️ Its mission? To run sets of commands, perform backups, and safely tuck those backups into S3-compatible storage. Whether you're backing up your pet project or enterprise data, poc-gocron offers an easy, customizable solution for all your backup needs.

## ✨ Features

- **⏰ Scheduled Backups**: Set and forget with cron syntax to automate backups.
- **🛠 Flexible Script Execution**: Runs your custom scripts to get those backups ready.
- **🌍 Environment and Custom Variable Support**: Leverages both environment and specially crafted variables within your scripts.
- **🔄 S3-Compatible Uploads**: Sends your backups straight to any S3-compatible storage.
- **🐳 Docker Support**: Ready to sail within Docker containers for consistent and isolated backup environments.

## 📋 Requirements

- **Go Environment**: Ready to build and run the application.
- **S3-Compatible Storage**: Must have the necessary access credentials.
- **Docker Environment (Optional)**: For those who prefer a containerized deployment.

## ⚙️ Configuration

### 🌐 Environment Variables

Set up these environment variables to get started:

```env
S3_ENDPOINT=your_s3_endpoint  # URL to your S3-compatible storage
S3_REGION=your_s3_region      # The storage region
S3_BUCKET=your_bucket_name    # The name of the bucket where backups will be stored
S3_SECRET_KEY=your_secret_key # Your S3 storage secret key
S3_ACCESS_KEY=your_access_key # Your S3 storage access key
CONFIG_PATH=path_to_config.yml # Path to the backup configuration file
S3_AUTO_CREATE_BUCKET=true or false # Whether to create the bucket if it doesn't exist
```

### 🗂 Backup Configuration (config.yml)

Craft a `config.yml` in your root directory or specified `CONFIG_PATH` to define your backup jobs. Peek at [config.example.yaml](./config.example.yml) for a sample setup!

### 🐳 Docker Usage

Kick off with this Dockerfile, prepped with essential tools (e.g., SQL clients) for your backup journey:

```dockerfile
FROM ghcr.io/siddhant-k-code/poc-gocron:main
ENV TZ="Asia/Kolkata"
COPY config.yml config.yml
```

Feel free to beef up your Dockerfile with extras like rsync, ssh, etc., depending on your backup needs.

### 🚢 Running the Docker Container

Fire up your Docker container with all the needed environment variables:

```bash
docker run -e S3_ENDPOINT=... -e S3_REGION=... -e S3_BUCKET=... \
  -e S3_SECRET_KEY=... -e S3_ACCESS_KEY=... -e CONFIG_PATH=config.yml \
  -e S3_AUTO_CREATE_BUCKET=true --name my-backuper my-backup-image
```

## 🛠 Build and Installation

Build it straight from the source:

```bash
go build -o poc-gocron .
```

Ready to run:

```bash
./poc-gocron
```

Make sure all your environment settings are dialed in before you hit go!

## 🎉 Conclusion

poc-gocron makes setting up and managing automated backups a breeze, safeguarding your data with ease. With Docker by its side and straightforward setup, it fits seamlessly into any workflow, ensuring your data's safety and your peace of mind. Happy backing up! 🎈
