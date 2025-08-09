# Creative Studio Video Server API 使用指南

## 服务器信息
- 运行地址: http://localhost:8080
- 状态: ✅ 运行中

## API 端点

### 1. 健康检查
```bash
curl http://localhost:8080/health
```

### 2. 上传视频
```bash
curl -X POST \
  http://localhost:8080/api/v1/videos/upload \
  -F "video=@/path/to/your/video.mp4"
```

### 3. 列出上传的文件
```bash
curl http://localhost:8080/api/v1/videos/files
```

### 4. 视频拼接
```bash
curl -X POST \
  http://localhost:8080/api/v1/videos/concatenate \
  -H "Content-Type: application/json" \
  -d '{
    "files": ["video1.mp4", "video2.mp4"],
    "output_name": "merged_video.mp4",
    "quality": "medium"
  }'
```

### 5. 下载拼接后的视频
```bash
curl http://localhost:8080/api/v1/videos/download/merged_video.mp4 \
  -o merged_video.mp4
```

### 6. 获取视频信息
```bash
curl http://localhost:8080/api/v1/videos/info/your_video.mp4
```

### 7. 删除文件
```bash
# 删除上传的文件
curl -X DELETE http://localhost:8080/api/v1/videos/your_video.mp4

# 删除输出文件
curl -X DELETE http://localhost:8080/api/v1/videos/merged_video.mp4?type=output
```

### 8. 列出输出文件
```bash
curl http://localhost:8080/api/v1/videos/output
```

## 功能特性
- ✅ 视频上传
- ✅ 视频拼接 (使用 FFmpeg)
- ✅ 文件管理
- ✅ 本地存储
- ❌ 无需数据库
- ❌ 无需用户认证
- ❌ 无需任务队列

## 目录结构
```
uploads/     # 上传的视频文件
output/      # 拼接后的视频文件
```

## 注意事项
1. 确保系统已安装 FFmpeg
2. 上传文件大小限制为 500MB
3. 支持的视频格式: MP4, AVI, MOV, MKV, WEBM, FLV, WMV, M4V
4. 服务器需要在 uploads 和 output 目录有读写权限