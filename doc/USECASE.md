# 📌使用过程中的最佳实践实例
## nacos的启动脚本修改
 👉 末尾启动代码的替换

```sh
...
# start
# We use exec to replace the shell process with the Java process,
# so that process managers like procmate can track the correct PID.
exec $JAVA ${JAVA_OPT}
```