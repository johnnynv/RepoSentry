#!/bin/bash

# RepoSentry Pipeline Naming Utilities
# 实现方案 A: 压缩版本命名方案

# 字符标准化函数
sanitize_name() {
    local input="$1"
    # 将特殊字符替换为连字符
    echo "$input" | sed 's/[\/_.:]/-/g' | sed 's/--*/-/g' | tr '[:upper:]' '[:lower:]'
}

# 智能截断函数
smart_truncate() {
    local input="$1"
    local max_length="$2"
    
    if [ ${#input} -le $max_length ]; then
        echo "$input"
    else
        # 保留前缀，不加省略号以避免特殊字符
        echo "${input:0:$max_length}"
    fi
}

# 从 URL 提取 Git 类型和仓库信息
parse_repo_url() {
    local repo_url="$1"
    
    if [[ "$repo_url" == *"github.com"* ]]; then
        echo "gh"
    elif [[ "$repo_url" == *"gitlab.com"* ]]; then
        echo "gl"
    elif [[ "$repo_url" == *"gitea"* ]]; then
        echo "gt"
    elif [[ "$repo_url" == *"bitbucket"* ]]; then
        echo "bb"
    else
        echo "git"
    fi
}

# 从完整仓库名称提取 owner 和 repo
parse_repo_fullname() {
    local repo_fullname="$1"
    local owner=$(echo "$repo_fullname" | cut -d'/' -f1)
    local repo=$(echo "$repo_fullname" | cut -d'/' -f2-)
    
    echo "$owner|$repo"
}

# 生成 Bootstrap PipelineRun 名称
generate_bootstrap_name() {
    local repo_url="$1"
    local repo_fullname="$2"
    local repo_branch="$3"
    local commit_sha="$4"
    
    local git_type=$(parse_repo_url "$repo_url")
    local repo_info=$(parse_repo_fullname "$repo_fullname")
    local owner=$(sanitize_name "$(echo "$repo_info" | cut -d'|' -f1)")
    local repo=$(sanitize_name "$(echo "$repo_info" | cut -d'|' -f2)")
    local branch=$(sanitize_name "$repo_branch")
    local commit7=$(echo "$commit_sha" | cut -c1-7)
    
    # 固定长度分配策略 - 确保总长度 ≤ 63
    # "reposentry-bootstrap-" (20) + git_type (2) + commit7 (7) + 分隔符 (4) = 33
    # 剩余 30 字符分配给 owner(8) + repo(8) + branch(6) = 22，还剩 8 字符缓冲
    
    branch=$(smart_truncate "$branch" 6)
    owner=$(smart_truncate "$owner" 8)
    repo=$(smart_truncate "$repo" 8)
    
    # 构建名称: reposentry-bootstrap-{type}-{owner}-{repo}-{branch}-{commit7}
    local name="reposentry-bootstrap-${git_type}-${owner}-${repo}-${branch}-${commit7}"
    
    # 最终安全检查，如果仍然超长则进一步截断
    if [ ${#name} -gt 63 ]; then
        # 进一步截断
        branch=$(smart_truncate "$branch" 4)
        owner=$(smart_truncate "$owner" 6)
        repo=$(smart_truncate "$repo" 6)
        name="reposentry-bootstrap-${git_type}-${owner}-${repo}-${branch}-${commit7}"
    fi
    
    echo "$name"
}

# 生成用户 PipelineRun 名称
generate_user_pipeline_name() {
    local pipeline_name="$1"
    local repo_url="$2"
    local repo_fullname="$3"
    local repo_branch="$4"
    local commit_sha="$5"
    
    local git_type=$(parse_repo_url "$repo_url")
    local repo_info=$(parse_repo_fullname "$repo_fullname")
    local owner=$(sanitize_name "$(echo "$repo_info" | cut -d'|' -f1)")
    local repo=$(sanitize_name "$(echo "$repo_info" | cut -d'|' -f2)")
    local branch=$(sanitize_name "$repo_branch")
    local commit7=$(echo "$commit_sha" | cut -c1-7)
    
    # 固定长度分配策略 - 确保总长度 ≤ 63
    # "-auto-" (5) + git_type (2) + commit7 (7) + 分隔符 (4) = 18
    # 剩余 45 字符分配给 pipeline(12) + owner(8) + repo(8) + branch(6) = 34，还剩 11 字符缓冲
    
    pipeline_name=$(smart_truncate "$pipeline_name" 12)
    owner=$(smart_truncate "$owner" 8)
    repo=$(smart_truncate "$repo" 8)
    branch=$(smart_truncate "$branch" 6)
    
    # 构建名称: {pipeline}-auto-{type}-{owner}-{repo}-{branch}-{commit7}
    local name="${pipeline_name}-auto-${git_type}-${owner}-${repo}-${branch}-${commit7}"
    
    # 最终安全检查，如果仍然超长则进一步截断
    if [ ${#name} -gt 63 ]; then
        # 进一步截断
        pipeline_name=$(smart_truncate "$pipeline_name" 8)
        owner=$(smart_truncate "$owner" 6)
        repo=$(smart_truncate "$repo" 6)
        branch=$(smart_truncate "$branch" 4)
        name="${pipeline_name}-auto-${git_type}-${owner}-${repo}-${branch}-${commit7}"
    fi
    
    echo "$name"
}

# 生成丰富的标签信息
generate_labels() {
    local repo_url="$1"
    local repo_fullname="$2"
    local repo_branch="$3"
    local commit_sha="$4"
    local pipeline_name="$5"
    local trigger_type="$6"  # bootstrap-pipeline 或 auto
    
    local git_type=$(parse_repo_url "$repo_url")
    local repo_info=$(parse_repo_fullname "$repo_fullname")
    local owner="$(echo "$repo_info" | cut -d'|' -f1)"
    local repo="$(echo "$repo_info" | cut -d'|' -f2)"
    local commit7=$(echo "$commit_sha" | cut -c1-7)
    
    cat << LABELS_EOF
    reposentry.io/git-type: "$git_type"
    reposentry.io/owner: "$owner"
    reposentry.io/repo: "$repo"
    reposentry.io/branch: "$repo_branch"
    reposentry.io/commit-sha: "$commit7"
    reposentry.io/commit-full: "$commit_sha"
    reposentry.io/trigger-type: "$trigger_type"
LABELS_EOF

    if [ -n "$pipeline_name" ]; then
        echo "    reposentry.io/pipeline: \"$pipeline_name\""
    fi
}

