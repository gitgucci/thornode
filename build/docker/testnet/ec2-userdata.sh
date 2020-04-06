#!/usr/bin/env bash

export THORNODE_REPO="https://gitlab.com/thorchain/thornode.git"
export BRANCH=bifrost-daily-churning
export GIT_PATH=/opt/thornode
export LOGFILE=/var/log/thornode.log
export THORNODE_ENV=testnet
export SEED_ENDPOINT=https://${THORNODE_ENV}-seed.thorchain.info

# install essential packages
echo "installing essential packages"
apt-get update -y
apt-get install -y \
    build-essential \
    jq \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg2 \
    cron \
    software-properties-common \
    unzip \
    python3-pip

systemctl enable cron # enable cron

echo "install aws cli"
export LC_ALL=C
pip3 install --upgrade pip
pip3 install awscli --upgrade
export THORNODE_PASSWD=$(aws secretsmanager get-secret-value --secret-id ${THORNODE_ENV}-signer-passwd --region us-east-1  | jq -r .SecretString | awk -F'[:]' '{print $2}' | sed -e 's/}//' | sed -e 's/"//g')

# install docker-compose
echo "installing docker-compose" >> $LOGFILE
curl -L "https://github.com/docker/compose/releases/download/1.25.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
docker-compose version >> $LOGFILE

# clone and start thornode
echo "cloning thornode" >> $LOGFILE
rm -rf $GIT_PATH
mkdir -p $GIT_PATH
git clone --single-branch --branch ${BRANCH} ${THORNODE_REPO} ${GIT_PATH} >> $LOGFILE

mkdir -p /opt/${THORNODE_ENV}
chmod -R 777 /opt/${THORNODE_ENV}

cat <<EOF > /opt/${THORNODE_ENV}/binance-bootstrap
#!/bin/sh

start_stack () {
    cd $GIT_PATH/build/docker
    make run-${THORNODE_ENV}-binance >> $LOGFILE
}

start_stack
sleep 120
EOF

cat <<EOF > /opt/${THORNODE_ENV}/standalone-bootstrap
#!/bin/sh

start_stack () {
    cd $GIT_PATH/build/docker
    export TAG=${THORNODE_ENV} && \
    export SIGNER_PASSWD=${THORNODE_PASSWD} && \
    export BINANCE_HOST="http://testnet-binance.thorchain.info:26657" && \
    make run-${THORNODE_ENV}-standalone >> $LOGFILE
}

start_stack
sleep 120
EOF

cat <<EOF > /opt/${THORNODE_ENV}/churn-bootstrap
#!/bin/sh

start_stack () {
    cd $GIT_PATH/build/docker
    export TAG=${THORNODE_ENV} && \
    export SIGNER_PASSWD=${THORNODE_PASSWD} && \
    export BINANCE_HOST="http://testnet-binance.thorchain.info:26657" && \
    export PEER=$(curl -s ${SEED_ENDPOINT}/bonded_nodes.json |tail -n 1 |jq '.'ip | sed -e 's/"//g' -e "s/null//g") && \
    make run-${THORNODE_ENV}-validator >> $LOGFILE
}

# install binance cli
echo "installing bnbcli >> $LOGFILE"

wget https://media.githubusercontent.com/media/binance-chain/node-binary/master/cli/testnet/0.6.2/linux/tbnbcli
chmod +x tbnbcli
mv tbnbcli /usr/local/bin/.
tbnbcli version >> $LOGFILE

start_stack
sleep 120
EOF