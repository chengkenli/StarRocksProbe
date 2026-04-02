// ==================== 分页相关全局变量和函数 ====================
let queryAllData = []; // 存储从后端获取的全部数据
let queryCurrentPage = 1; // 当前页码
let queryPageSize = 20; // 每页显示条数，可根据需要调整
let queryCurrentSearchTerm = ""; // 搜索条件变量

// 获取当前页的数据（支持搜索过滤）
function getqueryCurrentPageData() {
    let dataToShow = queryAllData;

    // 如果存在搜索条件，先进行过滤
    if (queryCurrentSearchTerm) {
        dataToShow = queryAllData.filter(item => {
            return Object.values(item).some(value => 
                String(value).toLowerCase().includes(queryCurrentSearchTerm.toLowerCase())
            );
        });
    }

    // 计算分页
    const startIndex = (queryCurrentPage - 1) * queryPageSize;
    const endIndex = startIndex + queryPageSize;
    return dataToShow.slice(startIndex, endIndex);
}

// 计算总页数
function getTotalPages() {
    let dataToShow = queryAllData;
    if (queryCurrentSearchTerm) {
        dataToShow = queryAllData.filter(item => {
            return Object.values(item).some(value => 
                String(value).toLowerCase().includes(queryCurrentSearchTerm.toLowerCase())
            );
        });
    }
    return Math.ceil(dataToShow.length / queryPageSize);
}

// 跳转到指定页面
function goToQueryPage(page) {
    const totalPages = getTotalPages();
    if (page < 1) page = 1;
    if (page > totalPages) page = totalPages;

    queryCurrentPage = page;
    renderTable(getqueryCurrentPageData());
    updatePaginationControls();
}

// 改变每页显示条数
function changequeryPageSize(newSize) {
    queryPageSize = parseInt(newSize);
    queryCurrentPage = 1; // 重置到第一页
    renderTable(getqueryCurrentPageData());
    updatePaginationControls();
}




// 生成页码按钮
function generateQueryPageNumbers(totalPages) {
    let pageNumbers = '';
    const maxVisiblePages = 5;

    let startPage = Math.max(1, queryCurrentPage - Math.floor(maxVisiblePages / 2));
    let endPage = Math.min(totalPages, startPage + maxVisiblePages - 1);

    if (endPage - startPage + 1 < maxVisiblePages) {
        startPage = Math.max(1, endPage - maxVisiblePages + 1);
    }

    for (let i = startPage; i <= endPage; i++) {
        pageNumbers += `
            <li class="page-item ${i === queryCurrentPage ? 'active' : ''}">
                <a class="page-link" href="#" onclick="goToQueryPage(${i})">${i}</a>
            </li>
        `;
    }

    return pageNumbers;
}

// 更新分页控件状态 - 修复版本
function updatePaginationControls() {
    let dataToShow = queryAllData;
    if (queryCurrentSearchTerm) {
        dataToShow = queryAllData.filter(item => {
            return Object.values(item).some(value => 
                String(value).toLowerCase().includes(queryCurrentSearchTerm.toLowerCase())
            );
        });
    }
    
    const totalItems = dataToShow.length;
    const totalPages = Math.ceil(totalItems / queryPageSize);
    const startItem = totalItems === 0 ? 0 : (queryCurrentPage - 1) * queryPageSize + 1;
    const endItem = Math.min(queryCurrentPage * queryPageSize, totalItems);

    // 更新分页控件内容
    const paginationContainer = document.getElementById('pagination-controls');
    if (paginationContainer) {
        paginationContainer.innerHTML = `
            <div class="text-muted small">
                显示第 ${startItem} 到 ${endItem} 条，共 ${totalItems} 条记录
            </div>
            <nav>
                <ul class="pagination pagination-sm mb-0">
                    <li class="page-item ${queryCurrentPage === 1 ? 'disabled' : ''}">
                        <a class="page-link" href="#" onclick="goToQueryPage(1)">首页</a>
                    </li>
                    <li class="page-item ${queryCurrentPage === 1 ? 'disabled' : ''}">
                        <a class="page-link" href="#" onclick="goToQueryPage(${queryCurrentPage - 1})">上一页</a>
                    </li>

                    ${generateQueryPageNumbers(totalPages)}

                    <li class="page-item ${queryCurrentPage === totalPages ? 'disabled' : ''}">
                        <a class="page-link" href="#" onclick="goToQueryPage(${queryCurrentPage + 1})">下一页</a>
                    </li>
                    <li class="page-item ${queryCurrentPage === totalPages ? 'disabled' : ''}">
                        <a class="page-link" href="#" onclick="goToQueryPage(${totalPages})">末页</a>
                    </li>
                </ul>
            </nav>
            <div class="d-flex align-items-center">
                <label class="form-label mb-0 me-2 small">每页显示：</label>
                <select class="form-select form-select-sm" style="width: auto;" onchange="changequeryPageSize(this.value)">
                    <option value="20" ${queryPageSize === 20 ? 'selected' : ''}>20条</option>
                    <option value="50" ${queryPageSize === 50 ? 'selected' : ''}>50条</option>
                    <option value="100" ${queryPageSize === 100 ? 'selected' : ''}>100条</option>
                </select>
            </div>
        `;
    }
}

// 创建分页控件HTML - 修复版本
function createPaginationControls() {
    // 检查是否已存在分页控件，避免重复创建
    if (document.getElementById('pagination-controls')) {
        return;
    }

    const dataToShow = queryAllData;
    const totalItems = dataToShow.length;
    const totalPages = Math.ceil(totalItems / queryPageSize);

    // 创建分页容器
    const paginationContainer = document.createElement("div");
    paginationContainer.id = "pagination-controls";
    paginationContainer.className = "d-flex justify-content-between align-items-center mt-3 p-2 bg-light border-top";

    // 将分页控件放在表格容器的最后面
    const tableContainer = document.querySelector('.table-responsive');
    if (tableContainer) {
        tableContainer.appendChild(paginationContainer);
    }

    // 分页信息
    const startItem = totalItems === 0 ? 0 : (queryCurrentPage - 1) * queryPageSize + 1;
    const endItem = Math.min(queryCurrentPage * queryPageSize, totalItems);

    paginationContainer.innerHTML = `
        <div class="text-muted small">
            显示第 ${startItem} 到 ${endItem} 条，共 ${totalItems} 条记录
        </div>
        <nav>
            <ul class="pagination pagination-sm mb-0">
                <li class="page-item ${queryCurrentPage === 1 ? 'disabled' : ''}">
                    <a class="page-link" href="#" onclick="goToQueryPage(1)">首页</a>
                </li>
                <li class="page-item ${queryCurrentPage === 1 ? 'disabled' : ''}">
                    <a class="page-link" href="#" onclick="goToQueryPage(${queryCurrentPage - 1})">上一页</a>
                </li>

                ${generateQueryPageNumbers(totalPages)}

                <li class="page-item ${queryCurrentPage === totalPages ? 'disabled' : ''}">
                    <a class="page-link" href="#" onclick="goToQueryPage(${queryCurrentPage + 1})">下一页</a>
                </li>
                <li class="page-item ${queryCurrentPage === totalPages ? 'disabled' : ''}">
                    <a class="page-link" href="#" onclick="goToQueryPage(${totalPages})">末页</a>
                </li>
            </ul>
        </nav>
        <div class="d-flex align-items-center">
            <label class="form-label mb-0 me-2 small">每页显示：</label>
            <select class="form-select form-select-sm" style="width: auto;" onchange="changequeryPageSize(this.value)">
                <option value="20" ${queryPageSize === 20 ? 'selected' : ''}>20条</option>
                <option value="50" ${queryPageSize === 50 ? 'selected' : ''}>50条</option>
                <option value="100" ${queryPageSize === 100 ? 'selected' : ''}>100条</option>
            </select>
        </div>
    `;
    
    console.log('分页控件创建完成');
}




// ==================== 统一搜索功能 ====================
// 搜索功能初始化
function initializeQueryUnifiedSearch() {
    const searchBox = document.getElementById('search');

    // 从本地存储恢复搜索状态
    const savedSearch = localStorage.getItem('queryMonitorSearch');
    if (savedSearch) {
        searchBox.value = savedSearch;
        queryCurrentSearchTerm = savedSearch; // 修复：使用正确的变量名
        filterUnifiedTable(savedSearch);
    }

    // 添加搜索事件监听
    searchBox.addEventListener('input', function() {
        const searchTerm = this.value.trim();
        queryCurrentSearchTerm = searchTerm; // 修复：使用正确的变量名

        // 保存搜索状态到本地存储
        localStorage.setItem('queryMonitorSearch', searchTerm);

        filterUnifiedTable(searchTerm);
    });

    // 添加清除搜索功能
    searchBox.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            this.value = '';
            queryCurrentSearchTerm = ''; // 修复：使用正确的变量名
            localStorage.removeItem('queryMonitorSearch');
            filterUnifiedTable('');
        }
    });
}

// 统一表格过滤函数 - 同时搜索query-data和resource-data
function filterUnifiedTable(searchTerm) {
    // 更新查询数据的搜索条件
    queryCurrentSearchTerm = searchTerm;
    queryCurrentPage = 1; // 搜索后回到第一页

    // 过滤查询数据表格
    filterQueryTable(searchTerm);

    // 过滤资源数据表格
    filterResourceTable(searchTerm);
}

// 查询数据表格过滤
function filterQueryTable(searchTerm) {
    queryCurrentSearchTerm = searchTerm;
    renderTable(getqueryCurrentPageData());
}

// 资源数据表格过滤
function filterResourceTable(searchTerm) {
    const tableBody = document.getElementById('resource-data');
    if (!tableBody) return;

    const rows = tableBody.getElementsByTagName('tr');
    let visibleCount = 0;

    // 如果没有搜索词，显示所有行
    if (!searchTerm) {
        for (let i = 0; i < rows.length; i++) {
            rows[i].style.display = '';
            visibleCount++;
        }
        removeNoResultsMessage('resource');
        return;
    }

    const searchTermLower = searchTerm.toLowerCase();

    // 遍历所有行进行过滤
    for (let i = 0; i < rows.length; i++) {
        const row = rows[i];
        const cells = row.getElementsByTagName('td');
        let rowContainsSearchTerm = false;

        // 检查每个单元格的内容
        for (let j = 0; j < cells.length; j++) {
            const cellText = getTextContent(cells[j]);
            if (cellText.toLowerCase().includes(searchTermLower)) {
                rowContainsSearchTerm = true;
                break;
            }
        }

        // 根据搜索结果显示/隐藏行
        if (rowContainsSearchTerm) {
            row.style.display = '';
            visibleCount++;
            highlightText(row, searchTerm);
        } else {
            row.style.display = 'none';
        }
    }

    // 如果没有匹配的结果，显示提示信息
    if (visibleCount === 0) {
        showNoResultsMessage(searchTerm, 'resource');
    } else {
        removeNoResultsMessage('resource');
    }
}

// 获取元素的纯文本内容（不包含HTML标签）
function getTextContent(element) {
    const temp = document.createElement('div');
    temp.innerHTML = element.innerHTML;
    return temp.textContent || temp.innerText || '';
}

// 高亮匹配的文本
function highlightText(row, searchTerm) {
    const cells = row.getElementsByTagName('td');
    const searchTermLower = searchTerm.toLowerCase();

    for (let i = 0; i < cells.length; i++) {
        const cell = cells[i];
        const originalHTML = cell.innerHTML;

        const regex = new RegExp(`(${escapeRegExp(searchTerm)})`, 'gi');
        const highlightedHTML = originalHTML.replace(regex, '<mark class="search-highlight">$1</mark>');

        cell.innerHTML = highlightedHTML;
    }
}

// 转义正则表达式特殊字符
function escapeRegExp(string) {
    return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// 显示无结果消息
function showNoResultsMessage(searchTerm, tableType) {
    const tableBody = document.getElementById(`${tableType}-data`);
    if (!tableBody) return;

    // 检查是否已经显示了无结果消息
    if (tableBody.querySelector('.no-results-message')) {
        return;
    }

    const noResultsRow = document.createElement('tr');
    noResultsRow.className = 'no-results-message';
    noResultsRow.innerHTML = `
        <td colspan="${tableType === 'query' ? '9' : '8'}" class="text-center py-4 text-muted">
            <i class="bi bi-search me-2"></i>没有匹配的数据</td>
    `;

    tableBody.appendChild(noResultsRow);
}

// 移除无结果消息
function removeNoResultsMessage(tableType) {
    const tableBody = document.getElementById(`${tableType}-data`);
    if (!tableBody) return;

    const noResultsMessage = tableBody.querySelector('.no-results-message');
    if (noResultsMessage) {
        noResultsMessage.remove();
    }
}

// ==================== 原有的业务函数 ====================

function fetchData() {
  return apiFetch("/query").then((data) => {
    // 保存全部数据到全局变量
    queryAllData = data.data || [];
    // 重置到第一页
    queryCurrentPage = 1;
    // 渲染当前页数据
    renderTable(getqueryCurrentPageData());
    updateCounters(data);
    // 更新分页控件
    updatePaginationControls();
    return data;
  });
}

// api.js - 确保函数是全局的
window.fetchData = function () {
    return apiFetch("/query")
        .then((data) => {
            // 保存全部数据到全局变量
            queryAllData = data.data || [];
            
            // 重置到第一页
            const totalPages = Math.ceil(queryAllData.length / queryPageSize);
            if (queryCurrentPage > totalPages && totalPages > 0) {
              queryCurrentPage = totalPages;
            } else if (totalPages === 0) {
              queryCurrentPage = 1;
            }
            
            // 渲染当前页数据
            renderTable(getqueryCurrentPageData());
            updateCounters(data);
            // 确保分页控件已创建并更新状态
            createPaginationControls();
            updatePaginationControls();
            
            return data;
        })
        .catch((error) => {
            console.error("获取数据失败:", error);
        });
};

function renderTable(data) {
  const tableBody = document.getElementById("query-data");

  tableBody.innerHTML = "";

  if (!data || data.length === 0) {
    const row = document.createElement("tr");
    row.innerHTML = `<td colspan="9" class="text-center py-2 text-muted">${queryCurrentSearchTerm ? '没有匹配的数据' : '没有数据'}</td>`;
    tableBody.appendChild(row);
    // 确保分页控件状态更新
    updatePaginationControls();
    return;
  }

  data.forEach((item, index) => {
    const row = document.createElement("tr");
    let timeClass = "";
    let timeText = item.time;
    if (item.time > 1500) {
      timeClass = "critical-query-cell";
      timeText += ' <span class="badge bg-danger">高消耗</span>';
    } else if (item.time > 600) {
      timeClass = "slow-query-cell";
      timeText += ' <span class="badge bg-warning text-dark">慢查询</span>';
    } else if (item.time > 300) {
      timeClass = "slow-query-cell";
      timeText += ' <span class="badge bg-info text-dark">慢查询</span>';
    } else if (item.time === 0 && item.user === "cndlopsns") {
      timeClass = "slow-query-cell";
      timeText += ' <span class="badge bg-info text-white">connection</span>';
    }

    // 处理用户角色显示
    let userDisplay = item.userurl;
    let roleTexts = [];

    // 收集所有角色文本
    if (item.admin) roleTexts.push('管理员');
    if (item.point) roleTexts.push('二级保障');
    if (item.shortlist) roleTexts.push('短查询保障');
    if (item.whitelist) roleTexts.push('白名单');

    // 将所有角色文本合并到一个span中
    if (roleTexts.length > 0) {
      userDisplay += ` <span class="badge bg-white text-success">${roleTexts.join(' | ')}</span>`;
    }

    // 处理追踪用户名
    let clientDisplay = item.host;
    if (item.clientuser !== null && item.clientuser !== undefined) {
      clientDisplay += `<span class="badge bg-white text-dark border border-gray-300 px-2 py-1">${item.clientuser}</span>`;
    }
    if (item.host && item.host.includes("10.9.")) {
      clientDisplay += `<span class="badge bg-white text-dark border border-gray-300 px-2 py-1">Support Center VM Containers</span>`;
    }
    // 处理stmt内容，用于tooltip显示
    let stmtContent = item.info || "无";
    if (stmtContent.length > 200) {
      stmtContent = stmtContent.substring(0, 200) + "...";
    }

    // 修改：计算实际序号（全局序号，而不是当前页的）
    const actualIndex = (queryCurrentPage - 1) * queryPageSize + index;

    row.innerHTML = `
                    <td class="px-2 py-1 text-center"><span class="badge text-bg-primary rounded-1 fw-bold">${index + 1}</span></td>
                    <td class="py-1" data-tooltip="${stmtContent}">${item.id}</td>
					<td class="py-1" data-tooltip="${item.feip}">${userDisplay}</td>
					<td class="py-1" data-tooltip="${item.OperationalMsg}">${item.Operational}</td>
                    <td class="py-1" data-tooltip="${item.ctxip}">${clientDisplay}</td>   
                    <td class="py-1 ${timeClass}" data-tooltip="${item.gethour}">${timeText}</td> 
                    <td class="py-1 text-center">${
                      item.isPending
                        ? '<i data-tooltip="pending" class="bi bi-hourglass-top text-danger"></i>'
                        : '<i data-tooltip="running" class="bi bi-check-circle text-success"></i>'
                    }</td>
                    <td class="py-1" data-tooltip="disconnect ${item.user} all connections">${item.warehouse}</td>
					<td class="py-1" data-tooltip="kill Pid">${item.command}</td>
                `;

    if (item.time > 1200) {
        // 添加进度条效果
        row.classList.add('highlight-row');
        // 初始化
        updateProgress(row, 0);
    }

    tableBody.appendChild(row);
  });

  // 确保分页控件状态更新
  updatePaginationControls();
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    // 初始化搜索功能
    initializeQueryUnifiedSearch();

    // 立即创建分页控件，不再延迟
    createPaginationControls();
    
    // 初始加载数据
    if (typeof fetchData === 'function') {
        fetchData();
    }
});