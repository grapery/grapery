#!/bin/bash

# Grapery 容器部署脚本
# 使用方法: ./deploy.sh [start|stop|restart|logs|status]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目配置
PROJECT_NAME="grapery"
COMPOSE_FILE="docker-compose.yml"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Docker和Docker Compose
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    log_success "依赖检查完成"
}

# 创建必要的目录和文件
setup_environment() {
    log_info "设置环境..."
    
    # 创建SSL目录
    mkdir -p ssl
    
    # 创建日志目录
    mkdir -p logs
    
    # 设置文件权限
    chmod +x deploy.sh
    
    log_success "环境设置完成"
}

# 构建镜像
build_images() {
    log_info "构建Docker镜像..."
    
    docker-compose -f $COMPOSE_FILE build --no-cache
    
    log_success "镜像构建完成"
}

# 启动服务
start_services() {
    log_info "启动服务..."
    
    docker-compose -f $COMPOSE_FILE up -d
    
    log_success "服务启动完成"
    
    # 等待服务启动
    log_info "等待服务启动..."
    sleep 10
    
    # 检查服务状态
    check_services_status
}

# 停止服务
stop_services() {
    log_info "停止服务..."
    
    docker-compose -f $COMPOSE_FILE down
    
    log_success "服务停止完成"
}

# 重启服务
restart_services() {
    log_info "重启服务..."
    
    docker-compose -f $COMPOSE_FILE down
    docker-compose -f $COMPOSE_FILE up -d
    
    log_success "服务重启完成"
    
    # 等待服务启动
    log_info "等待服务启动..."
    sleep 10
    
    # 检查服务状态
    check_services_status
}

# 查看日志
show_logs() {
    log_info "显示服务日志..."
    
    docker-compose -f $COMPOSE_FILE logs -f
}

# 查看特定服务日志
show_service_logs() {
    local service_name=$1
    log_info "显示 $service_name 服务日志..."
    
    docker-compose -f $COMPOSE_FILE logs -f $service_name
}

# 检查服务状态
check_services_status() {
    log_info "检查服务状态..."
    
    docker-compose -f $COMPOSE_FILE ps
    
    # 检查容器健康状态
    echo ""
    log_info "容器健康状态:"
    
    local containers=("grapery-grapes" "grapery-syncworker" "grapery-mcps" "grapery-vippay" "grapery-mysql" "grapery-redis" "grapery-nginx")
    
    for container in "${containers[@]}"; do
        if docker ps --format "table {{.Names}}\t{{.Status}}" | grep -q "$container"; then
            local status=$(docker ps --format "table {{.Names}}\t{{.Status}}" | grep "$container" | awk '{print $2}')
            log_success "$container: $status"
        else
            log_error "$container: 未运行"
        fi
    done
}

# 清理资源
cleanup() {
    log_info "清理资源..."
    
    # 停止并删除容器
    docker-compose -f $COMPOSE_FILE down
    
    # 删除未使用的镜像
    docker image prune -f
    
    # 删除未使用的网络
    docker network prune -f
    
    log_success "资源清理完成"
}

# 备份数据
backup_data() {
    log_info "备份数据..."
    
    local backup_dir="backup/$(date +%Y%m%d_%H%M%S)"
    mkdir -p $backup_dir
    
    # 备份MySQL数据
    docker exec grapery-mysql mysqldump -u root -proot123 grapery > $backup_dir/mysql_backup.sql
    
    # 备份Redis数据
    docker exec grapery-redis redis-cli BGSAVE
    sleep 2
    docker cp grapery-redis:/data/dump.rdb $backup_dir/redis_backup.rdb
    
    log_success "数据备份完成: $backup_dir"
}

# 恢复数据
restore_data() {
    local backup_dir=$1
    
    if [ -z "$backup_dir" ]; then
        log_error "请指定备份目录"
        exit 1
    fi
    
    if [ ! -d "$backup_dir" ]; then
        log_error "备份目录不存在: $backup_dir"
        exit 1
    fi
    
    log_info "恢复数据..."
    
    # 恢复MySQL数据
    if [ -f "$backup_dir/mysql_backup.sql" ]; then
        docker exec -i grapery-mysql mysql -u root -proot123 grapery < $backup_dir/mysql_backup.sql
        log_success "MySQL数据恢复完成"
    fi
    
    # 恢复Redis数据
    if [ -f "$backup_dir/redis_backup.rdb" ]; then
        docker cp $backup_dir/redis_backup.rdb grapery-redis:/data/dump.rdb
        docker exec grapery-redis redis-cli BGREWRITEAOF
        log_success "Redis数据恢复完成"
    fi
}

# 显示帮助信息
show_help() {
    echo "Grapery 容器部署脚本"
    echo ""
    echo "使用方法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  start          启动所有服务"
    echo "  stop           停止所有服务"
    echo "  restart        重启所有服务"
    echo "  logs           查看所有服务日志"
    echo "  logs [服务名]  查看特定服务日志"
    echo "  status         查看服务状态"
    echo "  build          构建Docker镜像"
    echo "  cleanup        清理资源"
    echo "  backup         备份数据"
    echo "  restore [目录] 恢复数据"
    echo "  help           显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 start"
    echo "  $0 logs grapes"
    echo "  $0 backup"
    echo "  $0 restore backup/20231201_120000"
}

# 主函数
main() {
    local command=$1
    
    case $command in
        "start")
            check_dependencies
            setup_environment
            start_services
            ;;
        "stop")
            stop_services
            ;;
        "restart")
            check_dependencies
            restart_services
            ;;
        "logs")
            local service_name=$2
            if [ -n "$service_name" ]; then
                show_service_logs $service_name
            else
                show_logs
            fi
            ;;
        "status")
            check_services_status
            ;;
        "build")
            check_dependencies
            build_images
            ;;
        "cleanup")
            cleanup
            ;;
        "backup")
            backup_data
            ;;
        "restore")
            local backup_dir=$2
            restore_data $backup_dir
            ;;
        "help"|"--help"|"-h"|"")
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@" 