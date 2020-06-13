#!/usr/bin/env bash

# bento/centos-8 で VM を作成した場合の初期構築

if ! rpm -qa | grep php ; then
    echo "  - install php"
    # dnf install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
    # dnf install -y https://rpms.remirepo.net/enterprise/remi-release-8.rpm
    # dnf module install -y php:remi-7.4
    dnf -y install php php-mbstring php-intl php-xml php-json php-pdo php-mysqlnd
fi

# if ! rpm -qa | grep nginx ; then
#     echo "  - install nginx"
#     dnf install -y nginx
# fi

if ! rpm -qa | grep mysql-server ; then
    echo "  - install mysql"
    dnf install -y @mysql:8.0
    echo "" >> /etc/my.cnf.d/mysql-server.cnf
    echo "collation-server=utf8mb4_bin" >> /etc/my.cnf.d/mysql-server.cnf
    # MySQL8.0 だと utf8mb4_bin は使用できないらしい
    # https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_collation_for_utf8mb4
    # echo "default_collation_for_utf8mb4=utf8mb4_bin" >> /etc/my.cnf.d/mysql-server.cnf
    systemctl start mysqld
    systemctl enable mysqld
    mysql -e "create user odysseus@localhost identified by 'odysseus';"
    mysql -e "grant all privileges on *.* to odysseus@localhost;"
    mysql -e "create user odysseus@'%' identified by 'odysseus';"
    mysql -e "grant all privileges on *.* to odysseus@'%';"
    mysql -e "create database odysseus;"
    # mysql -Dodysseus < /vagrant_data/initial.sql
    cd /vagrant_data/myapp/; php artisan migrate
fi

timedatectl set-timezone Asia/Tokyo
