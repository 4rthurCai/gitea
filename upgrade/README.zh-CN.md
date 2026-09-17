# Gitea fork 升级记录

目标：官方稳定版 **1.27.3**（2026-08-29 发布）。

- 上游标签：`v1.27.3`，提交 `146cc3eec57174711eac0e0a0c7b38670c6e3922`。
- 原 fork：`origin/release/v1.22.0`，提交 `29f023ef40b7f24c7b9a61ab31f47eef336b6d46`。
- 原上游基线：`v1.22.0`，提交 `803b0c9ab43f809e47f42fad951e7f5fb6652e42`。
- 升级分支：`upgrade/gitea-1.27.3`。原 fork 分支保留。
- 服务器尚未确定，未部署，也未连接或迁移生产数据库。

## 必须保留的定制

| 定制                 | 迁移检查点                                                            |
| -------------------- | --------------------------------------------------------------------- |
| jAccount OAuth       | provider、图标、账号/邮箱/姓名/学号映射及首次注册自动提交             |
| 隐藏 HTTP clone      | 保留 SSH clone 按钮，适配新版 DOM                                     |
| 用户资料限制         | 禁止修改姓名、邮箱隐私和位置；禁止自行删除账户                        |
| Wiki 活动            | 独立 Wiki 统计、作者头像和图表；适配新版仓库存储接口及 Vue/TypeScript |
| CompanyStaff pretend | 保留公司团队选择和原有身份切换逻辑、`COMPANY_TEAM_NAME` 配置          |
| 屏蔽用户限制         | 保留 fork 对普通用户的限制和原有组织规则                              |
| 通知                 | 排除归档仓库的问题通知，并保留额外索引                                |
| Issue/PR 排序        | 默认按最近更新；显式选择其他排序仍可用                                |
| Actions              | 保留额外索引、tag badge、badge message API 和 `no status` 响应        |
| API 对象格式         | 原 fork 修复已由新版的 `ObjectFormatName` 转换覆盖                    |

跨版本迁移需要调整内部接口和模板，不能直接保持旧源代码逐字不变；定制业务行为是保留目标。

## 构建

使用项目 `go.mod` 和 `package.json` 指定的版本：Go 1.26.4+、Node 22.18+、pnpm 11.9.0。

```sh
pnpm install --frozen-lockfile
GITEA_VERSION=1.27.3 TAGS='bindata timetzdata' make build
./gitea --version
```

Linux 服务器应使用与目标架构匹配的构建产物。也可从**此 fork 工作树**构建自定义镜像：

```sh
docker build --build-arg GITEA_VERSION=1.27.3 -t gitea-fork:1.27.3 .
```

必须使用包含定制的自建产物；官方现成二进制或镜像不包含本 fork 功能。Docker 构建尚需在目标环境验证。

## 服务器确定后

1. 确认 SSH、操作系统/架构、Docker 或 systemd、Gitea 运行用户、代理和端口、数据库类型及版本、配置与全部存储路径。
2. 保存当前二进制/镜像、配置、密钥和服务定义。停止写入后备份数据库、Git/Wiki 仓库、LFS、附件、包、Actions 数据以及外部对象存储；先验证备份能恢复。
3. 将备份恢复到隔离测试环境，禁用向生产发送邮件、webhook 和 runner 作业，使用自建升级产物启动并检查数据库迁移日志。
4. 验证登录、Git SSH clone/push、Issue/PR、权限、Actions，以及上表每项定制。jAccount 回调地址和测试账号需要实际环境配合。
5. 演练成功后安排维护窗口，停止生产写入、重新备份、替换为验证过的产物，检查迁移日志与健康状态后开放访问。
6. 若失败，停止新版服务，恢复同一时间点的旧数据库、文件数据、配置与旧程序。**已迁移数据库不能直接交给旧版程序运行。**

升级与恢复依据：

- https://blog.gitea.com/release-of-1.27.3/
- https://docs.gitea.com/installation/upgrade-from-gitea/
- https://docs.gitea.com/administration/backup-and-restore/

## 本地验证结果

- `make frontend`：通过。
- `pnpm exec vue-tsc --noEmit`：通过。
- 修改的前端文件 ESLint：通过。
- `GITEA_VERSION=1.27.3 TAGS='bindata timetzdata' make build`：通过，生成本地 macOS/arm64 程序 `gitea`，包含前端资源。
- jAccount、用户权限/资料、活动统计、组织、Issue 索引、模板、Actions 的相关 Go 测试：通过（DB 索引子包无测试）。
- `make generate-swagger`：通过，包含 badge message API 的 Swagger/OpenAPI 定义。
- `GITEA_TEST_DATABASE=sqlite go test ./tests/integration -run '^TestForkBadgeMessage$' -count=1`：通过；验证令牌要求和默认分支、branch、tag 的无运行记录响应。
- `git diff --check v1.27.3`：通过。
- Elasticsearch / Meilisearch 测试因本机未启动相应外部服务失败；未据此改动产品行为。
- 本次未执行生产数据迁移、真实 jAccount 登录和目标服务器 Docker 构建。

构建期间本机 Go 模块缓存曾出现解压不完整的问题；修复后使用独立 `GOCACHE=/tmp/gitea-1.27.3-go-cache` 完成验证。项目 `go.mod`、`go.sum` 保持官方 1.27.3 的版本。
