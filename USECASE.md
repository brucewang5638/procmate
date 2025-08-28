## 📌 建议使用示例(仅限SMZJG项目)

1. 确保项目路径是/opt/hk/

   1. 项目路径不符合：可用kill所有进程后， mv /opt/hk  /opt/hk/smzjg

2. 确保其内部文件是hkservice和component两个文件夹

3. 将整个脚本文件放到 /opt/hk/script

4. chmod +x -R /opt/hk/script 赋予可执行权限

5. 将 hk-smzjg-components.service 和hk-smzjg-services.service 放到 /etc/systemd/system/中

   1. 防止后执行 systemctl daemon-reload

6. systemctl start hk-smzjg-components  

   systemctl status hk-smzjg-components 

   systemctl start hk-smzjg-services
   systemctl status hk-smzjg-services

7. journalctl -u ${服务名} -f

   例如： journalctl -u hk-smzjg-components.service -f

8. 拷贝 hk-smzjg-service hk-smzjg-component 命令

   sudo ln -s /opt/hk/smzjg/script/service/hk-smzjg-service.sh /usr/local/bin/hk-smzjg-service
   sudo ln -s /opt/hk/smzjg/script/component/hk-smzjg-component.sh /usr/local/bin/hk-smzjg-component
