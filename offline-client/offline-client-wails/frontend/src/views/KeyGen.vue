<template>
    <div class="page task-entry-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">密钥生成</h2>
                <p class="page-subtitle">优先导入在线系统导出的托管钱包生成任务包，再发起 2-of-3 等 MPC keygen。</p>
            </div>
            <el-button type="primary" icon="el-icon-upload2" @click="$router.push('/offline-tasks')">
                导入 JSON 任务包
            </el-button>
        </div>

        <el-card>
            <el-alert
                type="info"
                :closable="false"
                title="keygen 主流程已经收敛到“离线任务”页面。">
            </el-alert>

            <el-steps :active="0" simple class="steps">
                <el-step title="导入 offline_task JSON"></el-step>
                <el-step title="选择管理员/警员参与方"></el-step>
                <el-step title="警员确认并执行 MPC"></el-step>
                <el-step title="下载 offline_result JSON"></el-step>
            </el-steps>

            <el-descriptions :column="1" border>
                <el-descriptions-item label="任务包类型">custody_keygen</el-descriptions-item>
                <el-descriptions-item label="结果包类型">custody_keygen_result</el-descriptions-item>
                <el-descriptions-item label="推荐文件名">offline_result_&lt;task_no&gt;.json</el-descriptions-item>
            </el-descriptions>
        </el-card>
    </div>
</template>

<script>
export default {
    name: 'KeyGen'
}
</script>

<style scoped>
.steps {
    margin: 18px 0;
}
</style>
