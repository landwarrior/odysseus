#!/usr/bin/env bash

# bento/almalinux-9 で VM を作成した場合の初期構築

timedatectl set-timezone Asia/Tokyo

# 補完してくれるやつ
if ! [[ $(rpm -qa | grep bash-completion) ]] ; then
    echo "  - dnf install bash-completion"
    sudo dnf install -y bash-completion
fi

if ! [[ $(rpm -qa | grep unzip) ]] ; then
  echo "  - install zip unzip"
  dnf install -y zip unzip
fi

if ! [[ $(rpm -qa | grep ^git) ]] ; then
  echo "  - install git"
  dnf install -y git
fi

if ! [[ $(rpm -qa | grep php) ]] ; then
  echo "  - install php"
  dnf install -y https://rpms.remirepo.net/enterprise/remi-release-8.rpm
  dnf module install -y php:remi-7.4
  dnf install -y php php-intl php-pdo php-mysqlnd
fi

if ! [[ $(find / -name composer 2>/dev/null) ]] ; then
  echo "  - install composer"
  php -r "copy('https://getcomposer.org/installer', 'composer-setup.php');"
  php -r "if (hash_file('sha384', 'composer-setup.php') === 'dac665fdc30fdd8ec78b38b9800061b4150413ff2e3b6f88543c636f7cd84f6db9189d43a81e5503cda447da73c7e5b6') { echo 'Installer verified'; } else { echo 'Installer corrupt'; unlink('composer-setup.php'); } echo PHP_EOL;"
  # Installer corrupt が出た場合は以下のサイトを参照してハッシュ値を変更しないといけない
  # https://getcomposer.org/download/
  php composer-setup.php
  php -r "unlink('composer-setup.php');"
  mv composer.phar /usr/local/bin/composer
fi

if ! [[ $(rpm -qa | grep nginx | grep -v filesystem) ]] ; then
    echo "  - install nginx"
    dnf install -y nginx
fi

if ! [[ $(rpm -qa | grep npm) ]] ; then
  echo "  - install npm"
  dnf install -y npm
fi

# chef インストール
if [[ $(rpm -qa | grep chef) ]] ; then
    echo "  * skip installing chef"
else
    echo "  - chef installing"
    curl -L https://omnitruck.chef.io/install.sh | sudo bash -s -- -v 18.3.0
fi

echo ""

# chef 実行
sudo echo yes | chef-client -z -c /vagrant_data/chef-repo/solo.rb -j /vagrant_data/chef-repo/nodes/odysseus.json

cd /vagrant_data/myapp/; php artisan migrate


# このあとやらなければいけないこと
# /vagrant_data/fileput.sh
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
