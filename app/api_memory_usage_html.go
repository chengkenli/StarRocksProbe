/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    api_memory_usage_html
 *@date    2025/12/31 15:25
 */

package app

import (
	"StarRocksProbe/tools"
	"fmt"
)

func memory_usage_file(filename, data string) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>星辰守望 - StarRocks小助手</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }
        
        body {
            background-color: #f5f7fa;
            color: #333;
            padding: 20px;
        }
        
        .header {
            text-align: center;
            margin-bottom: 30px;
            padding: 20px;
            background: linear-gradient(135deg, #1e3c72, #2a5298);
            color: white;
            border-radius: 10px;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
        }
        
        .header h1 {
            font-size: 2.5rem;
            margin-bottom: 10px;
        }
        
        .header p {
            font-size: 1.1rem;
            opacity: 0.9;
        }
        
        .container {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .server-card {
            background: white;
            border-radius: 10px;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
            overflow: hidden;
            transition: transform 0.3s ease, box-shadow 0.3s ease;
        }
        
        .server-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 8px 16px rgba(0, 0, 0, 0.12);
        }
        
        .server-header {
            background: #2a5298;
            color: white;
            padding: 15px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        
        .server-ip {
            font-weight: bold;
            font-size: 1.2rem;
        }
        
        .server-status {
            background: #4CAF50;
            padding: 5px 10px;
            border-radius: 20px;
            font-size: 0.9rem;
        }
        
        .memory-summary {
            padding: 15px;
            border-bottom: 1px solid #eee;
        }
        
        .memory-bar {
            height: 10px;
            background: #e0e0e0;
            border-radius: 5px;
            margin: 10px 0;
            overflow: hidden;
        }
        
        .memory-fill {
            height: 100%%;
            background: linear-gradient(90deg, #4CAF50, #8BC34A);
            border-radius: 5px;
            transition: width 0.5s ease;
        }
        
        .memory-info {
            display: flex;
            justify-content: space-between;
            font-size: 0.9rem;
            color: #666;
        }
        
        .memory-details {
            padding: 15px;
        }
        
        .detail-item {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px dashed #eee;
        }
        
        .detail-label {
            font-weight: 500;
        }
        
        .detail-value {
            font-family: 'Courier New', monospace;
        }
        
        .footer {
            text-align: center;
            margin-top: 30px;
            padding: 20px;
            color: #666;
            font-size: 0.9rem;
        }
        
        @media (max-width: 768px) {
            .container {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>星辰守望 - StarRocks小助手</h1>
        <p>BE组件内存使用情况（静态）</p>
    </div>
    
    <div class="container" id="server-container">
        <!-- 服务器卡片将通过JavaScript动态生成 -->
    </div>
    
    <div class="footer">
        <p>最后更新: <span id="last-update"></span></p>
        <p>采集系统: 星辰守望 - StarRocks小助手</p>
    </div>

    <script>
        // 内存监控数据
        const memoryData = %v;

        // 将内存大小转换为字节
        function parseMemorySize(size) {
            if (size === "none" || size === "0") return 0;
            
            const units = {
                'K': 1024,
                'M': 1024 * 1024,
                'G': 1024 * 1024 * 1024
            };
            
            const unit = size.slice(-1);
            const value = parseFloat(size.slice(0, -1));
            
            return value * (units[unit] || 1);
        }

        // 格式化内存大小显示
        function formatMemorySize(bytes) {
            if (bytes === 0) return "0";
            
            const units = ['B', 'K', 'M', 'G'];
            let unitIndex = 0;
            let value = bytes;
            
            while (value >= 1024 && unitIndex < units.length - 1) {
                value /= 1024;
                unitIndex++;
            }
            
            return value.toFixed(unitIndex > 0 ? 2 : 0) + units[unitIndex];
        }

        // 计算内存使用百分比
        function calculateUsagePercentage(current, limit) {
            if (limit === 0 || limit === "none") return 0;
            
            const currentBytes = parseMemorySize(current);
            const limitBytes = parseMemorySize(limit);
            
            if (limitBytes === 0) return 0;
            
            return Math.min(100, (currentBytes / limitBytes) * 100);
        }

        // 生成服务器卡片
        function generateServerCards() {
            const container = document.getElementById('server-container');
            container.innerHTML = '';
            
            memoryData.forEach(server => {
                const processRecord = server.MemoryRecord.find(record => record.level === "1" && record.label === "process");
                
                if (!processRecord) return;
                
                const usagePercentage = calculateUsagePercentage(
                    processRecord.current_consumption, 
                    processRecord.limit
                );
                
                const card = document.createElement('div');
                card.className = 'server-card';
                
                card.innerHTML = %s
	<div class="server-header">
	<div class="server-ip">${server.IP}</div>
	<div class="server-status">运行中</div>
	</div>
	<div class="memory-summary">
	<h3>进程内存使用情况</h3>
	<div class="memory-bar">
	<div class="memory-fill" style="width: ${usagePercentage}%%"></div>
	</div>
	<div class="memory-info">
	<span>当前: ${processRecord.current_consumption}</span>
	<span>峰值: ${processRecord.peak_consumption}</span>
	<span>限制: ${processRecord.limit}</span>
	</div>
	</div>
	<div class="memory-details">
	<h4>内存组件详情</h4>
	${server.MemoryRecord
		.filter(record => record.level === "2")
		.map(record => %s
                                <div class="detail-item">
                                    <span class="detail-label">${record.label}</span>
                                    <span class="detail-value">${record.current_consumption} / ${record.limit === "none" ? "无限制" : record.limit}</span>
                                </div>
                            %s).join('')}
	</div>
		%s;
                
                container.appendChild(card);
            });
        }

        // 更新最后更新时间
        function updateLastUpdateTime() {
            const now = new Date();
            document.getElementById('last-update').textContent = now.toLocaleString();
        }

        // 页面加载完成后初始化
        document.addEventListener('DOMContentLoaded', function() {
            generateServerCards();
            updateLastUpdateTime();
            
            // 每30秒更新一次数据（模拟实时更新）
            setInterval(updateLastUpdateTime, 30000);
        });
    </script>
</body>
</html>`, data, "`", "`", "`", "`")
	tools.WriteFile(filename, html)
}
