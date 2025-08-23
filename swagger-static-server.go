package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// 设置静态文件目录
	fs := http.FileServer(http.Dir("docs"))

	// 创建多路复用器
	mux := http.NewServeMux()

	// 添加根路径重定向
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// 重定向到swagger UI
			http.Redirect(w, r, "/swagger", http.StatusMovedPermanently)
			return
		}
		// 其他路径提供静态文件服务
		fs.ServeHTTP(w, r)
	})

	// 添加CORS支持
	mux.HandleFunc("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 读取并返回swagger.yaml文件
		content, err := os.ReadFile("docs/swagger.yaml")
		if err != nil {
			http.Error(w, "Failed to read swagger.yaml", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/yaml")
		w.Write(content)
	})

	// 添加一个简单的HTML页面来展示Swagger UI
	mux.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>RepoSentry API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/swagger.yaml',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "Standalone"
            });
        };
    </script>
</body>
</html>`

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	port := ":8081"
	fmt.Printf("🚀 Swagger静态服务器启动在 http://localhost%s\n", port)
	fmt.Printf("📖 Swagger UI: http://localhost%s/swagger\n", port)
	fmt.Printf("📄 YAML文件: http://localhost%s/swagger.yaml\n", port)
	fmt.Printf("📋 JSON文件: http://localhost%s/swagger.json\n", port)
	fmt.Printf("📁 静态文件: http://localhost%s/\n", port)
	fmt.Println("\n按 Ctrl+C 停止服务器")

	log.Fatal(http.ListenAndServe(port, mux))
}
