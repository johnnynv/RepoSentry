package api

// @title RepoSentry API
// @version 1.0
// @description A lightweight, cloud-native sentinel for monitoring GitLab and GitHub repositories
// @description This API provides endpoints for managing repository monitoring, viewing status, and accessing metrics.

// @contact.name RepoSentry Support
// @contact.url https://github.com/johnnynv/RepoSentry
// @contact.email support@reposentry.dev

// @license.name MIT
// @license.url https://github.com/johnnynv/RepoSentry/blob/main/LICENSE

// @host localhost:8080
// @BasePath /

// @schemes http https

// @tag.name Health
// @tag.description Health check and readiness endpoints

// @tag.name Status
// @tag.description System status and runtime information

// @tag.name Repositories
// @tag.description Repository management and monitoring

// @tag.name Events
// @tag.description Event history and processing

// @tag.name Metrics
// @tag.description Application metrics and statistics

// @tag.name System
// @tag.description System information and version

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Bearer token for API authentication
