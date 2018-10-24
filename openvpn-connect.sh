#!/usr/bin/env bash

log_stdout=true
log_fileout=true
logfile="/data/log/openvpn/vpn-connect.log"




function log_warn() {
if [[ $log_stdout == "true" && $log_fileout == "true" ]];then
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[33m [WARN] $1\033[0m"
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[33m [WARN] $1\033[0m" >> ${logfile}
elif [[ $log_stdout == "true" && $log_fileout == "false" ]];then
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[33m [WARN] $1\033[0m"
elif [[ $log_stdout == "false" && $log_fileout == "true" ]];then
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[33m [WARN] $1\033[0m" >> ${logfile}
elif [[ $log_stdout == "false" && $log_fileout == "false" ]];then
echo "no log"
fi
}

function log_error() {
if [[ $log_stdout == "true" && $log_fileout == "true" ]];then
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[31m [ERROR] $1\033[0m"
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[31m [ERROR] $1\033[0m" >> ${logfile}
elif [[ $log_stdout == "true" && $log_fileout == "false" ]];then
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[31m [ERROR] $1\033[0m"
elif [[ $log_stdout == "false" && $log_fileout == "true" ]];then
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[31m [ERROR] $1\033[0m" >> ${logfile}
elif [[ $log_stdout == "false" && $log_fileout == "false" ]];then
echo "no log"
fi
}


function log_info() {
if [[ $log_stdout == "true" && $log_fileout == "true" ]];then
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[32m [INFO] $1\033[0m"
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[32m [INFO] $1\033[0m" >> ${logfile}
elif [[ $log_stdout == "true" && $log_fileout == "false" ]];then
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[32m [INFO] $1\033[0m"
elif [[ $log_stdout == "false" && $log_fileout == "true" ]];then
echo -e  $(date +%Y年%m月%d日-%H:%M:%S) "\033[32m [INFO] $1\033[0m" >> ${logfile}
elif [[ $log_stdout == "false" && $log_fileout == "false" ]];then
echo "no log"
fi
}


function main() {
if [[ $username == "" ]];then
log_error "没有获取到VPN的用户名"
exit 1
fi

log_info "==========================start================================="

log_info "登陆用户 $username 来自IP: $trusted_ip,分配的VPN IP为 $ifconfig_pool_remote_ip "

if [[ ! -f "/etc/openvpn/rules/$username" ]];then
    log_warn "不存在用户 $username 的规则"
else
    /usr/sbin/iptables -t filter -L $username
    if [[ $? -ne 0 ]];then
    log_warn "table filter chain  $username 不存在，开始创建chain $username"
    log_info "/usr/sbin/iptables -t filter -N $username"
    /usr/sbin/iptables -t filter -N $username
    else
    log_warn "table filter chain  $username 已经存在,清空链"
    log_info "/usr/sbin/iptables -F $username"
    /usr/sbin/iptables -F $username
    fi

    user_forward_rule_num=`/usr/sbin/iptables -L FORWARD |grep $username|wc -l`

    if [[ ${user_forward_rule_num} < 1 ]];then
    log_info "FORWARD链 里面没有用户$username 链的 放行规则，添加中..."
    log_info "/usr/sbin/iptables -I FORWARD -j $username"
    /usr/sbin/iptables -I FORWARD -j $username
    fi

    if [[ $username == ""  ]];then
    log_error "没有获取到变量 username "
    exit 2
    elif [[ $ifconfig_pool_remote_ip == "" ]];then
    log_error "没有获取到变量 ifconfig_pool_remote_ip"
    exit 3
    fi

    log_info "开始在 $username 链添加规则......"
    for line in `cat /etc/openvpn/rules/$username`
    do
        log_info "读取到文件记录是 $line"
        ip=`echo $line|cut -d "," -f1`
        port=`echo $line|cut -d "," -f2`
        proto=`echo $line|cut -d "," -f3`
        target=`echo $line|cut -d "," -f4`

        log_info "ip 是$ip ,端口是$port,协议是 $proto"
        if [[ $ip != "" && $port == "all" && $target != "" ]];then

        log_info "/usr/sbin/iptables -I $username -s $ifconfig_pool_remote_ip -d $ip -j $target"
        /usr/sbin/iptables -I $username -s $ifconfig_pool_remote_ip -d $ip -j $target
        elif [[ $ip != "" && $port != "all" && $port != "" && $proto != "" && $target != "" ]];then
        log_info "/usr/sbin/iptables -I $username -p $proto -s $ifconfig_pool_remote_ip -d $ip --dport $port -j $target"
        /usr/sbin/iptables -I $username -p $proto -s $ifconfig_pool_remote_ip -d $ip --dport $port -j $target
        else
        log_error "从 $username 文件中获取到的 格式规则不对，请检查文件"
        fi
    done
fi

log_info "==========================end================================="
}

main
