#!/bin/bash
# 阿里云统一平台迁移 - Git Worktree 环境初始化脚本
# 创建时间: 2026-02-02
# 用途: 为迁移任务创建独立的 worktree 工作目录

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
WORKTREE_ROOT="${HOME}/worktrees/video-dubbing"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# 分支定义
MAIN_BRANCH="feature/aliyun-unified-migration"
declare -a SUB_BRANCHES=(
    "feature/aliyun-asr-default"
    "feature/aliyun-oss-default"
    "feature/aliyun-llm-default"
    "feature/aliyun-tts-default"
    "feature/cleanup-legacy-services"
)

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查是否在 Git 仓库中
check_git_repo() {
    print_info "检查 Git 仓库状态..."

    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "当前目录不是 Git 仓库"
        exit 1
    fi

    print_success "Git 仓库检查通过"
}

# 检查工作区是否干净
check_clean_working_tree() {
    print_info "检查工作区状态..."

    if ! git diff-index --quiet HEAD -- 2>/dev/null; then
        print_warning "工作区有未提交的更改"
        read -p "是否继续? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "操作已取消"
            exit 0
        fi
    fi

    print_success "工作区检查通过"
}

# 创建 worktree 根目录
create_worktree_root() {
    print_info "创建 worktree 根目录: ${WORKTREE_ROOT}"

    if [ -d "${WORKTREE_ROOT}" ]; then
        print_warning "目录已存在: ${WORKTREE_ROOT}"
        read -p "是否删除并重新创建? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            print_info "删除现有目录..."
            rm -rf "${WORKTREE_ROOT}"
        else
            print_info "使用现有目录"
            return
        fi
    fi

    mkdir -p "${WORKTREE_ROOT}"
    print_success "目录创建成功"
}

# 创建主迁移分支
create_main_branch() {
    print_info "创建主迁移分支: ${MAIN_BRANCH}"

    # 检查分支是否已存在
    if git show-ref --verify --quiet "refs/heads/${MAIN_BRANCH}"; then
        print_warning "分支已存在: ${MAIN_BRANCH}"
        read -p "是否删除并重新创建? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            # 删除 worktree
            if [ -d "${WORKTREE_ROOT}/aliyun-migration" ]; then
                print_info "移除现有 worktree..."
                git worktree remove "${WORKTREE_ROOT}/aliyun-migration" --force || true
            fi
            # 删除分支
            print_info "删除现有分支..."
            git branch -D "${MAIN_BRANCH}" || true
        else
            print_info "跳过主分支创建"
            return
        fi
    fi

    # 创建 worktree
    print_info "创建 worktree: ${WORKTREE_ROOT}/aliyun-migration"
    git worktree add "${WORKTREE_ROOT}/aliyun-migration" -b "${MAIN_BRANCH}"

    # 复制迁移计划文档
    if [ -f "${PROJECT_ROOT}/docs/migration/ALIYUN_MIGRATION_PLAN.md" ]; then
        print_info "复制迁移计划文档到 worktree..."
        cp "${PROJECT_ROOT}/docs/migration/ALIYUN_MIGRATION_PLAN.md" \
           "${WORKTREE_ROOT}/aliyun-migration/docs/migration/"

        cd "${WORKTREE_ROOT}/aliyun-migration"
        git add docs/migration/ALIYUN_MIGRATION_PLAN.md
        git commit -m "docs: add aliyun unified migration plan" || true
        cd "${PROJECT_ROOT}"
    fi

    print_success "主迁移分支创建成功"
}

# 创建子任务分支
create_sub_branches() {
    print_info "创建子任务分支..."

    cd "${WORKTREE_ROOT}/aliyun-migration"

    local branch_names=(
        "asr-default"
        "oss-default"
        "llm-default"
        "tts-default"
        "cleanup-legacy"
    )

    for i in "${!SUB_BRANCHES[@]}"; do
        local branch="${SUB_BRANCHES[$i]}"
        local dir="${WORKTREE_ROOT}/${branch_names[$i]}"

        print_info "创建分支: ${branch}"

        # 检查分支是否已存在
        if git show-ref --verify --quiet "refs/heads/${branch}"; then
            print_warning "分支已存在: ${branch}"
            read -p "是否删除并重新创建? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                if [ -d "${dir}" ]; then
                    git worktree remove "${dir}" --force || true
                fi
                git branch -D "${branch}" || true
            else
                print_info "跳过: ${branch}"
                continue
            fi
        fi

        # 创建 worktree
        print_info "创建 worktree: ${dir}"
        git worktree add "${dir}" -b "${branch}"

        # 创建初始 commit
        cd "${dir}"

        # 创建任务说明文件
        local task_doc="docs/migration/TASK_${branch_names[$i]^^}.md"
        mkdir -p "$(dirname "${task_doc}")"

        cat > "${task_doc}" << EOF
# Task: ${branch}

## 目标
根据 ALIYUN_MIGRATION_PLAN.md 中的相关章节实施迁移

## 状态
- [ ] 开发中

## 负责人
TBD

## 进度
- [ ] 配置文件修改
- [ ] 代码实现
- [ ] 单元测试
- [ ] 集成测试
- [ ] 文档更新

## 相关链接
- [迁移计划](./ALIYUN_MIGRATION_PLAN.md)
- [主分支](${WORKTREE_ROOT}/aliyun-migration)

## 备注
EOF

        git add "${task_doc}"
        git commit -m "docs: add task documentation for ${branch}" || true

        print_success "分支创建成功: ${branch}"
    done

    cd "${PROJECT_ROOT}"
}

# 显示 worktree 列表
show_worktree_list() {
    print_info "\n当前 Worktree 列表:"
    echo "================================================"
    git worktree list
    echo "================================================"
}

# 创建快捷脚本
create_helper_scripts() {
    print_info "创建辅助脚本..."

    # 创建快速切换脚本
    cat > "${WORKTREE_ROOT}/goto.sh" << 'EOF'
#!/bin/bash
# 快速切换到指定 worktree 的脚本

case "$1" in
    main|migration|m)
        cd aliyun-migration
        ;;
    asr|a)
        cd asr-default
        ;;
    oss|o)
        cd oss-default
        ;;
    llm|l)
        cd llm-default
        ;;
    tts|t)
        cd tts-default
        ;;
    cleanup|c)
        cd cleanup-legacy
        ;;
    *)
        echo "用法: source goto.sh [main|asr|oss|llm|tts|cleanup]"
        echo "别名: m|a|o|l|t|c"
        return 1
        ;;
esac

echo "切换到: $(pwd)"
EOF
    chmod +x "${WORKTREE_ROOT}/goto.sh"

    # 创建状态检查脚本
    cat > "${WORKTREE_ROOT}/check_status.sh" << 'EOF'
#!/bin/bash
# 检查所有 worktree 的状态

DIRS=(
    "aliyun-migration"
    "asr-default"
    "oss-default"
    "llm-default"
    "tts-default"
    "cleanup-legacy"
)

for dir in "${DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo "========================================"
        echo "Worktree: $dir"
        echo "========================================"
        cd "$dir"
        echo "Branch: $(git branch --show-current)"
        echo "Status:"
        git status -s
        echo ""
        cd ..
    fi
done
EOF
    chmod +x "${WORKTREE_ROOT}/check_status.sh"

    # 创建批量提交脚本
    cat > "${WORKTREE_ROOT}/commit_all.sh" << 'EOF'
#!/bin/bash
# 批量提交所有 worktree 的更改

if [ -z "$1" ]; then
    echo "用法: ./commit_all.sh <commit_message>"
    exit 1
fi

COMMIT_MSG="$1"

DIRS=(
    "asr-default"
    "oss-default"
    "llm-default"
    "tts-default"
    "cleanup-legacy"
)

for dir in "${DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo "提交: $dir"
        cd "$dir"
        if ! git diff-index --quiet HEAD --; then
            git add .
            git commit -m "$COMMIT_MSG"
            git push origin "$(git branch --show-current)"
        else
            echo "  无更改，跳过"
        fi
        cd ..
    fi
done

echo "合并到主分支..."
cd aliyun-migration
for dir in "${DIRS[@]}"; do
    branch="feature/aliyun-${dir%-default}"
    [ "$dir" = "cleanup-legacy" ] && branch="feature/cleanup-legacy-services"
    echo "合并: $branch"
    git merge --no-ff "$branch" -m "merge: integrate $branch into unified migration"
done
git push origin feature/aliyun-unified-migration

echo "所有更改已提交并合并"
EOF
    chmod +x "${WORKTREE_ROOT}/commit_all.sh"

    print_success "辅助脚本创建成功"
}

# 显示使用说明
show_usage_info() {
    cat << EOF

${GREEN}================================${NC}
${GREEN}Worktree 环境初始化完成！${NC}
${GREEN}================================${NC}

${BLUE}工作目录:${NC}
${WORKTREE_ROOT}

${BLUE}分支结构:${NC}
main
└── ${MAIN_BRANCH} (${WORKTREE_ROOT}/aliyun-migration)
    ├── feature/aliyun-asr-default (${WORKTREE_ROOT}/asr-default)
    ├── feature/aliyun-oss-default (${WORKTREE_ROOT}/oss-default)
    ├── feature/aliyun-llm-default (${WORKTREE_ROOT}/llm-default)
    ├── feature/aliyun-tts-default (${WORKTREE_ROOT}/tts-default)
    └── feature/cleanup-legacy-services (${WORKTREE_ROOT}/cleanup-legacy)

${BLUE}快速开始:${NC}
# 切换到指定 worktree
cd ${WORKTREE_ROOT}
source goto.sh asr      # 切换到 ASR 迁移任务
source goto.sh oss      # 切换到 OSS 迁移任务
source goto.sh llm      # 切换到 LLM 迁移任务
source goto.sh tts      # 切换到 TTS 迁移任务
source goto.sh cleanup  # 切换到清理任务
source goto.sh main     # 切换到主迁移分支

# 检查所有 worktree 状态
cd ${WORKTREE_ROOT}
./check_status.sh

# 批量提交所有更改
cd ${WORKTREE_ROOT}
./commit_all.sh "feat: implement feature X"

${BLUE}开发流程:${NC}
1. 在各自的 worktree 中并行开发
2. 定期运行 ./commit_all.sh 提交并合并到主分支
3. 在主迁移分支运行集成测试
4. 所有测试通过后合并到 main

${BLUE}文档:${NC}
查看详细迁移计划:
  ${WORKTREE_ROOT}/aliyun-migration/docs/migration/ALIYUN_MIGRATION_PLAN.md

${YELLOW}注意事项:${NC}
- 每个 worktree 是独立的工作区，可以并行开发
- 定期将子分支合并到主迁移分支进行集成测试
- 提交前确保通过单元测试
- 遵循项目的代码规范和提交规范

EOF
}

# 主函数
main() {
    print_info "开始初始化 Git Worktree 环境..."
    echo ""

    cd "${PROJECT_ROOT}"

    check_git_repo
    check_clean_working_tree
    create_worktree_root
    create_main_branch
    create_sub_branches
    create_helper_scripts
    show_worktree_list
    show_usage_info

    print_success "\n初始化完成！"
}

# 执行主函数
main "$@"
