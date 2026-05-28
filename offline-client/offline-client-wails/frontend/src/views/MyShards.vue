<template>
    <div class="page my-shards-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">我的分片</h2>
                <p class="page-subtitle">查看当前账号参与生成并持有的离线密钥分片。</p>
            </div>
            <el-button icon="el-icon-refresh" :loading="loading" @click="loadShards">刷新</el-button>
        </div>

        <el-card>
            <el-table :data="shards" v-loading="loading" style="width: 100%">
                <el-table-column prop="address" label="地址" min-width="220"></el-table-column>
                <el-table-column prop="case_no" label="案件编号" width="150"></el-table-column>
                <el-table-column prop="task_no" label="任务编号" width="170"></el-table-column>
                <el-table-column prop="shard_index" label="分片序号" width="90"></el-table-column>
                <el-table-column label="门限" width="90">
                    <template slot-scope="scope">
                        {{ thresholdText(scope.row) }}
                    </template>
                </el-table-column>
                <el-table-column prop="record_id" label="Record ID" min-width="220"></el-table-column>
                <el-table-column prop="se_cplc" label="SE CPLC" min-width="220"></el-table-column>
                <el-table-column prop="status" label="状态" width="100">
                    <template slot-scope="scope">
                        <el-tag :type="statusTag(scope.row.status)">{{ scope.row.status }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="updated_at" label="更新时间" width="170">
                    <template slot-scope="scope">
                        {{ formatTime(scope.row.updated_at) }}
                    </template>
                </el-table-column>
            </el-table>

            <el-empty v-if="!loading && shards.length === 0" description="暂无分片记录" :image-size="90"></el-empty>
        </el-card>
    </div>
</template>

<script>
export default {
    name: 'MyShards',
    data() {
        return {
            loading: false,
            shards: []
        }
    },
    created() {
        this.loadShards()
    },
    methods: {
        async loadShards() {
            this.loading = true
            try {
                const response = await this.$offlineApi.listMyShards()
                this.shards = response.data.shards || []
            } catch (error) {
                this.$message.error(error.response?.data?.error || '查询我的分片失败')
            } finally {
                this.loading = false
            }
        },
        statusTag(status) {
            if (status === 'active') return 'success'
            if (status === 'destroyed') return 'danger'
            return 'warning'
        },
        thresholdText(row) {
            if (!row.required_signers || !row.total_parties) return '-'
            return `${row.required_signers} / ${row.total_parties}`
        },
        formatTime(value) {
            return value ? new Date(value).toLocaleString() : '-'
        }
    }
}
</script>
