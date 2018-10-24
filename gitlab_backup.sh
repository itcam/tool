#!/bin/bash
HDATE=gitlab_backup_`date +%Y%m%d_%H%m`.log
BACK_DIR="/var/opt/gitlab/backups"
BDATE=`date +%Y年%m月%d日%H:%m:%S`
SDATE=`date +%Y年%m月%d日%H:%m:%S`
/usr/bin/gitlab-rake gitlab:backup:create
more_num=`expr $(ls ${BACK_DIR} |grep "gitlab_backup.tar" |wc -l) - 30`
if [[ $more_num -gt 0 ]];then

    cd ${BACK_DIR}
    ls -lrt ${BACK_DIR} |grep "gitlab_backup.tar"| awk '{print $NF}' |head -${more_num} |xargs rm -fr
    echo "删除了${more_num}个备份文件" >> ${BACK_DIR}/backlog/${HDATE} 2>&1
else
    echo "没有需要删除的"
fi
