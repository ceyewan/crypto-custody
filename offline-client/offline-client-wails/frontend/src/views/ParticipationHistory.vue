<template>
    <div class="page participation-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">参与记录</h2>
                <p class="page-subtitle">只展示当前账号自己的 keygen、sign、移交、销毁参与记录。</p>
            </div>
            <el-button icon="el-icon-refresh" :loading="loading" @click="loadRecords">刷新</el-button>
        </div>

        <el-card>
            <el-table :data="records" v-loading="loading" style="width: 100%">
                <el-table-column prop="created_at" label="时间" width="180">
                    <template slot-scope="scope">
                        {{ formatTime(scope.row.created_at) }}
                    </template>
                </el-table-column>
                <el-table-column prop="type" label="类型" width="110">
                    <template slot-scope="scope">
                        <el-tag>{{ typeText(scope.row.type) }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="action" label="动作" min-width="180"></el-table-column>
                <el-table-column prop="resource_id" label="资源" min-width="180"></el-table-column>
                <el-table-column prop="result" label="结果" width="100">
                    <template slot-scope="scope">
                        <el-tag :type="scope.row.result === 'success' ? 'success' : 'danger'">
                            {{ scope.row.result }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="error_message" label="错误摘要" min-width="180"></el-table-column>
            </el-table>

            <el-empty v-if="!loading && records.length === 0" description="暂无参与记录" :image-size="90"></el-empty>
        </el-card>
    </div>
</template>

<script>
export default {
    name: 'ParticipationHistory',
    data() {
        return {
            records: [],
            loading: false
        }
    },
    created() {
        this.loadRecords()
    },
    methods: {
        async loadRecords() {
            this.loading = true
            try {
                const response = await this.$offlineApi.listMyParticipation(200)
                this.records = response.data.records || []
            } catch (error) {
                this.$message.error(error.response?.data?.error || '加载参与记录失败')
            } finally {
                this.loading = false
            }
        },
        typeText(type) {
            const map = {
                keygen: '私钥生成',
                sign: '交易签名',
                transfer: '私钥分片移交',
                destroy: '私钥销毁'
            }
            return map[type] || type
        },
        formatTime(value) {
            return value ? new Date(value).toLocaleString() : '-'
        }
    }
}
</script>
