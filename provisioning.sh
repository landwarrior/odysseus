#!/usr/bin/env bash

# bento/centos-8 で VM を作成した場合の初期構築

if ! rpm -qa | grep unzip ; then
  echo "  - install zip unzip"
  dnf install -y zip unzip p7zip
fi
# 以下は check error により入らないので今はできない
# if ! rpm -qa | grep gcc ; then
#   dnf groupinstall -y "Development Tools"
# fi

if ! rpm -qa | grep ^git ; then
  echo "  - install git"
  dnf install -y git
fi

if ! rpm -qa | grep php ; then
  echo "  - install php"
  dnf install -y https://rpms.remirepo.net/enterprise/remi-release-8.rpm
  dnf module install -y php:remi-7.4
  dnf install -y php php-intl php-pdo php-mysqlnd
fi

if ! find / -name composer 2>/dev/null ; then
  php -r "copy('https://getcomposer.org/installer', 'composer-setup.php');"
  php -r "if (hash_file('sha384', 'composer-setup.php') === 'e5325b19b381bfd88ce90a5ddb7823406b2a38cff6bb704b0acc289a09c8128d4a8ce2bbafcd1fcbdc38666422fe2806') { echo 'Installer verified'; } else { echo 'Installer corrupt'; unlink('composer-setup.php'); } echo PHP_EOL;"
  # Installer corrupt が出た場合は以下のサイトを参照してハッシュ値を変更しないといけない
  # https://getcomposer.org/download/
  php composer-setup.php
  php -r "unlink('composer-setup.php');"
  mv composer.phar /usr/local/bin/composer
fi

if ! rpm -qa | grep nginx | grep -v filesystem ; then
    echo "  - install nginx"
    dnf install -y nginx
fi

if ! rpm -qa | grep npm ; then
  echo "  - install npm"
  dnf install -y npm
fi

if ! rpm -qa | grep mysql-server ; then
  echo "  - install mysql"
  dnf install -y @mysql:8.0
  echo "collation-server=utf8mb4_bin" >> /etc/my.cnf.d/mysql-server.cnf
  # MySQL8.0 だと utf8mb4_bin は使用できないらしい
  # https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_default_collation_for_utf8mb4
  # echo "default_collation_for_utf8mb4=utf8mb4_bin" >> /etc/my.cnf.d/mysql-server.cnf
  systemctl start mysqld
  systemctl enable mysqld
  mysql -e "create user odysseus@localhost identified by 'odysseus';"
  mysql -e "create database odysseus;"
  mysql -e "grant all privileges on odysseus.* to odysseus@localhost;"
  # mysql -e "create user odysseus@'%' identified by 'odysseus';"
  # mysql -e "grant all privileges on odysseus.* to odysseus@'%';"
  # mysql -Dodysseus < /vagrant_data/initial.sql
  # cd /vagrant_data/myapp/; php artisan migrate
fi

timedatectl set-timezone Asia/Tokyo

# このあとやらなければいけないこと
# cd /var
# git clone https://github.com/landwarrior/odysseus.git
# cd /var/odysseus/myapp
# CRC のエラーが出ることがあるので、 php で composer を起動した方が確実かも
# php /usr/local/bin/composer install
# npm install
# npm run dev
# ここで .env を作成して修正する
# その際、 key を生成する
# php artisan key:generate
# php artisan migrate
# php /usr/local/bin/composer dump-autoload
# php artisan db:seed
# php-fpm の設定変更
# vi /etc/php-fpm.d/www.conf
#  user = nginx
#  group = nginx
#  listen = /var/run/php-fpm/php-fpm.sock
#  listen.owner = nginx
#  listen.group = nginx
#  ;listen.acl_users = apache,nginx
# nginx の設定変更(ググる)
# vi /etc/nginx/nginx.conf
# vi /etc/nginx/conf.d/php-fpm.conf
# acl はコメントアウト
# /var/odysseus/myapp/storage に権限がないといけない
# chmod 777 -R storage で丸っと付けちゃう
# nginx の 500 エラーの遷移をなくすと見えるようになる
