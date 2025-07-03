#!/bin/bash

# Grapery 容器部署脚本
# 使用方法: ./deploy.sh [start|stop|restart|logs|status|ssl]

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

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装"
        exit 1
    fi
    
    log_success "依赖检查完成"
}

# 启动服务
start_services() {
    log_info "启动服务..."
    
    docker-compose -f $COMPOSE_FILE up -d
    
    log_success "服务启动完成"
    
    # 等待服务启动
    log_info "等待服务启动..."
    sleep 15
    
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
    sleep 15
    
    # 检查服务状态
    check_services_status
}

# 查看日志
show_logs() {
    local service_name=$1
    if [ -n "$service_name" ]; then
        log_info "显示 $service_name 服务日志..."
        docker-compose -f $COMPOSE_FILE logs -f $service_name
    else
        log_info "显示所有服务日志..."
        docker-compose -f $COMPOSE_FILE logs -f
    fi
}

# 检查服务状态
check_services_status() {
    log_info "检查服务状态..."
    
    docker-compose -f $COMPOSE_FILE ps
    
    # 检查容器健康状态
    echo ""
    log_info "容器健康状态:"
    
    local containers=("grapery-grapes" "grapery-syncworker" "grapery-mcps" "grapery-vippay" "grapery-nginx" "grapery-mysql" "grapery-redis")
    
    for container in "${containers[@]}"; do
        if docker ps --format "table {{.Names}}\t{{.Status}}" | grep -q "$container"; then
            local status=$(docker ps --format "table {{.Names}}\t{{.Status}}" | grep "$container" | awk '{print $2}')
            log_success "$container: $status"
        else
            log_error "$container: 未运行"
        fi
    done
}

# 配置SSL证书
setup_ssl() {
    local domain_name=$1
    local email=$2
    
    if [ -z "$domain_name" ] || [ -z "$email" ]; then
        log_error "请提供域名和邮箱"
        echo "用法: $0 ssl <域名> <邮箱>"
        echo "示例: $0 ssl grapery.com admin@grapery.com"
        exit 1
    fi
    
    log_info "配置SSL证书..."
    
    # 停止nginx容器
    docker-compose -f $COMPOSE_FILE stop nginx
    
    # 申请SSL证书
    sudo certbot certonly --standalone \
        -d $domain_name \
        -d api.$domain_name \
        -d mcp.$domain_name \
        -d pay.$domain_name \
        --email $email \
        --agree-tos \
        --non-interactive
    
    # 设置证书权限
    sudo chown -R $USER:$USER /etc/letsencrypt
    
    # 重启nginx容器
    docker-compose -f $COMPOSE_FILE up -d nginx
    
    log_success "SSL证书配置完成！"
}

# 备份数据
backup_data() {
    log_info "备份数据..."
    
    local backup_dir="backup/$(date +%Y%m%d_%H%M%S)"
    mkdir -p $backup_dir
    
    # 备份MySQL数据
    docker exec grapery-mysql mysqldump -u root -p$MYSQL_ROOT_PASSWORD grapery > $backup_dir/mysql_backup.sql
    
    # 备份Redis数据
    docker exec grapery-redis redis-cli BGSAVE
    sleep 2
    docker cp grapery-redis:/data/dump.rdb $backup_dir/redis_backup.rdb
    
    # 备份配置文件
    cp .env $backup_dir/env_backup
    cp docker-compose.yml $backup_dir/docker-compose_backup.yml
    
    # 备份SSL证书
    sudo cp -r /etc/letsencrypt $backup_dir/ssl_backup
    
    log_success "数据备份完成: $backup_dir"
}

# 清理资源
cleanup() {
    log_info "清理资源..."
    
    # 停止并删除容器
    docker-compose -f $COMPOSE_FILE down
    
    # 删除未使用的镜像和网络
    docker system prune -f
    
    log_success "资源清理完成"
}

# 显示帮助信息
show_help() {
    echo "Grapery 容器部署脚本"
    echo ""
    echo "使用方法: $0 [命令] [参数]"
    echo ""
    echo "命令:"
    echo "  start          启动所有服务"
    echo "  stop           停止所有服务"
    echo "  restart        重启所有服务"
    echo "  logs [服务名]  查看服务日志"
    echo "  status         查看服务状态"
    echo "  ssl <域名> <邮箱> 配置SSL证书"
    echo "  backup         备份数据"
    echo "  cleanup        清理资源"
    echo "  help           显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 start"
    echo "  $0 logs grapes"
    echo "  $0 ssl grapery.com admin@grapery.com"
    echo "  $0 backup"
}

# 主函数
main() {
    local command=$1
    local arg1=$2
    local arg2=$3
    
    case $command in
        "start")
            check_dependencies
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
            show_logs $arg1
            ;;
        "status")
            check_services_status
            ;;
        "ssl")
            setup_ssl $arg1 $arg2
            ;;
        "backup")
            backup_data
            ;;
        "cleanup")
            cleanup
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