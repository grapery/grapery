-- 创建数据库
CREATE DATABASE IF NOT EXISTS grapery CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE grapery;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(50),
    avatar_url VARCHAR(255),
    phone VARCHAR(20),
    status ENUM('active', 'inactive', 'banned') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_email (email),
    INDEX idx_username (username)
);

-- 商品表
CREATE TABLE IF NOT EXISTS products (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'CNY',
    sku VARCHAR(50) UNIQUE NOT NULL,
    stock_quantity INT DEFAULT 0,
    status ENUM('active', 'inactive', 'out_of_stock') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_sku (sku),
    INDEX idx_status (status)
);

-- 订单表
CREATE TABLE IF NOT EXISTS orders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_no VARCHAR(50) UNIQUE NOT NULL,
    user_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    total_amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'CNY',
    status ENUM('pending', 'paid', 'cancelled', 'refunded') DEFAULT 'pending',
    payment_method VARCHAR(20),
    payment_provider VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (product_id) REFERENCES products(id),
    INDEX idx_order_no (order_no),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
);

-- 支付记录表
CREATE TABLE IF NOT EXISTS payment_records (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT NOT NULL,
    payment_no VARCHAR(50) UNIQUE NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'CNY',
    payment_method VARCHAR(20) NOT NULL,
    payment_provider VARCHAR(20) NOT NULL,
    status ENUM('pending', 'success', 'failed', 'cancelled') DEFAULT 'pending',
    transaction_id VARCHAR(100),
    callback_data JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    INDEX idx_payment_no (payment_no),
    INDEX idx_order_id (order_id),
    INDEX idx_status (status)
);

-- 订阅表
CREATE TABLE IF NOT EXISTS subscriptions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    subscription_no VARCHAR(50) UNIQUE NOT NULL,
    status ENUM('active', 'cancelled', 'expired', 'pending') DEFAULT 'pending',
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    auto_renew BOOLEAN DEFAULT TRUE,
    payment_method VARCHAR(20),
    payment_provider VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (product_id) REFERENCES products(id),
    INDEX idx_subscription_no (subscription_no),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
);

-- VIP用户表
CREATE TABLE IF NOT EXISTS vip_users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    subscription_id BIGINT,
    vip_level ENUM('basic', 'premium', 'enterprise') DEFAULT 'basic',
    status ENUM('active', 'inactive', 'expired') DEFAULT 'inactive',
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    remaining_credits INT DEFAULT 0,
    total_credits INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id),
    UNIQUE KEY unique_user_vip (user_id),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
);

-- 插入示例商品数据
INSERT INTO products (name, description, price, currency, sku, stock_quantity, status) VALUES
('基础VIP', '基础VIP会员服务，包含基础功能', 29.90, 'CNY', 'VIP_BASIC', 999999, 'active'),
('高级VIP', '高级VIP会员服务，包含更多功能', 99.90, 'CNY', 'VIP_PREMIUM', 999999, 'active'),
('企业VIP', '企业VIP会员服务，包含所有功能', 299.90, 'CNY', 'VIP_ENTERPRISE', 999999, 'active');

-- 创建触发器：订单状态更新时自动更新支付记录状态
DELIMITER //
CREATE TRIGGER update_payment_status_on_order_update
AFTER UPDATE ON orders
FOR EACH ROW
BEGIN
    IF NEW.status = 'paid' THEN
        UPDATE payment_records 
        SET status = 'success' 
        WHERE order_id = NEW.id AND status = 'pending';
    ELSEIF NEW.status = 'cancelled' THEN
        UPDATE payment_records 
        SET status = 'cancelled' 
        WHERE order_id = NEW.id AND status = 'pending';
    END IF;
END//
DELIMITER ; 