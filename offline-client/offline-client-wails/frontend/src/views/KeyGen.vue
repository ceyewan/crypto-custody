<template>
    <div class="page task-entry-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">生成私钥</h2>
                <p class="page-subtitle">优先导入在线系统导出的任务包，生成托管地址和对应的私钥分片。</p>
            </div>
            <el-button type="primary" icon="el-icon-upload2" @click="$router.push('/offline-tasks')">
                导入 JSON 任务包
            </el-button>
        </div>

        <el-card>
            <el-alert
                type="info"
                :closable="false"
                    title="生成私钥主流程已经收敛到“离线任务”页面。">
            </el-alert>

            <el-steps :active="0" simple class="steps">
                <el-step title="导入 offline_task JSON"></el-step>
                <el-step title="选择管理员/警员参与方"></el-step>
                    <el-step title="警员确认并生成私钥分片"></el-step>
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
