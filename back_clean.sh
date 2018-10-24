#!/usr/bin/env bash
#清理备份文件，只保留最新的7个
back_dir=/data/backup/
total_File_Num=`ls $back_dir |wc -l`
threshold=7
more_File_Num=$((${threshold}-${total_File_Num}))

if [[ $total_File_Num -gt $threshold ]];then
    echo "开始删除" 
    cd $back_dir
    ls $back_dir |head $more_File_Num |xargs rm -fr
fi
