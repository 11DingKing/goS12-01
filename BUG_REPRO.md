# Bug Reproduction

## 包的性质

当前 test_model_fix 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。要复现原始缺陷，必须检出下面固定的 parent SHA；不要在当前修复结果源码上期待重新出现修复前失败。生成系统使用的可信验证补丁和完整验证日志仅在本地留存，不提交到结果分支。

## 问题现象

电池舱抢修流程被卡住了，请帮我修复。

调度员派发抢修工单后，对应电池舱会被锁定并绑定到这张工单上。但只要舱体在这之后又上报了一次超温或绝缘不合格的测点数据，舱体状态就会从锁定退回告警，和工单的绑定关系一起丢掉；紧接着指派的工程师执行到场操作会被拒绝，提示舱体状态不满足开工条件。结果是工程师已经在现场却开不了工，整条抢修流程卡死。

期望行为：抢修工单在途期间（已派发、以及工程师到场作业中），舱体一直归属这张工单，新上报的测点数据要照常存下来并可读，但不改变舱体的检修状态；工程师到场、完工、班长与供电站双人验收要能正常走完，验收关单后舱体才回到正常状态并清空工单绑定。空闲舱体首次出现超温或绝缘异常时，仍然必须正常进入告警。

仓库里已有面向公开行为的测试覆盖上述场景。修复后请保证 go test ./... 全绿。

## 含 Bug 版本

- 仓库：11DingKing/goS12-01
- 仓库地址：https://github.com/11DingKing/goS12-01.git
- parent SHA：cea4d15d57e5ec13e5ed2796b31a9f7b0e86dbb3

## 复现步骤

```bash
git clone -- https://github.com/11DingKing/goS12-01.git bug-repro
cd bug-repro
git checkout --detach cea4d15d57e5ec13e5ed2796b31a9f7b0e86dbb3
go test ./internal/service -run "^TestReadingsDuringMaintenanceKeepCabinLock$|^TestReadingsOnNormalCabinStillRaiseAlarm$" -count=1 -v
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/service -run "^TestReadingsDuringMaintenanceKeepCabinLock$|^TestReadingsOnNormalCabinStillRaiseAlarm$" -count=1 -v
=== RUN   TestReadingsDuringMaintenanceKeepCabinLock
    cabin_readings_lock_test.go:58: readings arriving during a dispatched repair must not change the cabin status: want locked, got alarm
--- FAIL: TestReadingsDuringMaintenanceKeepCabinLock (0.00s)
=== RUN   TestReadingsOnNormalCabinStillRaiseAlarm
--- PASS: TestReadingsOnNormalCabinStillRaiseAlarm (0.00s)
FAIL
FAIL	batteryops/internal/service	0.025s
FAIL

```

stderr：

```text
(empty)
```

### linux/arm64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/service -run "^TestReadingsDuringMaintenanceKeepCabinLock$|^TestReadingsOnNormalCabinStillRaiseAlarm$" -count=1 -v
=== RUN   TestReadingsDuringMaintenanceKeepCabinLock
    cabin_readings_lock_test.go:58: readings arriving during a dispatched repair must not change the cabin status: want locked, got alarm
--- FAIL: TestReadingsDuringMaintenanceKeepCabinLock (0.00s)
=== RUN   TestReadingsOnNormalCabinStillRaiseAlarm
--- PASS: TestReadingsOnNormalCabinStillRaiseAlarm (0.00s)
FAIL
FAIL	batteryops/internal/service	0.001s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

定向验证通过：go test ./internal/service -run '^TestReadingsDuringMaintenanceKeepCabinLock$|^TestReadingsOnNormalCabinStillRaiseAlarm$' -count=1 -v
全量回归通过：go test -timeout=300s -count=1 ./...；go build ./... 与 go vet ./... 通过
linux/amd64 与 linux/arm64 两个架构均通过
不得修改或跳过测试；空闲舱体的正常告警路径必须保留
