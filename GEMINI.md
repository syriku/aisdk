# 基本项目规范

1. git提交消息的格式为 `[category] summary`。如果没有特别指定，不再后面添加详细解释。
    1. 具体存在的分类可以参考：chore, feature, bugfix, cleanup, refactor, build, pure ui
2. 代码中如果没有额外说明，不使用英文以外的字符。
3. 带有标签的分支（指子目录，例如feature/** prototype/**）进行合并（合并到main或相互之间合并）总是创建合并提交。合并提交的类型是分支的标签。
4. 不要把idea mcp的replace_text_in_file工具来当做修改文件的方法
5. 在任务修改完成后，调用语言对应的格式化工具格式化代码
