<template>
    <div class="notifications-container">
        <el-card>
            <div slot="header" class="clearfix">
                <span>通知消息</span>
                <el-button style="float: right; padding: 3px 0" type="text" @click="clearNotifications">
                    清空通知
                </el-button>
            </div>

            <el-table :data="notifications" style="width: 100%">
                <el-table-column prop="type" label="消息类型" width="180"></el-table-column>
                <el-table-column prop="timestamp" label="时间" width="180">
                    <template slot-scope="scope">
                        {{ new Date(scope.row.timestamp).toLocaleString() }}
                    </template>
                </el-table-column>
                <el-table-column label="内容">
                    <template slot-scope="scope">
                        <el-button type="text" @click="showMessageDetail(scope.row)">
                            查看详情
                        </el-button>
                    </template>
                </el-table-column>
            </el-table>

            <div v-if="notifications.length === 0" class="empty-state">
                暂无通知消息
            </div>
        </el-card>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'

export default {
    name: 'Notifications',
    computed: {
        ...mapGetters(['notifications'])
    },
    methods: {
        clearNotifications() {
            this.$store.commit('clearNotifications')
        },

        showMessageDetail(message) {
            this.$alert(JSON.stringify(message.content, null, 2), '消息详情', {
                closeOnClickModal: true
            })
        }
    }
}
</script>

<style scoped>
.notifications-container {
    padding: 20px;
}

.empty-state {
    text-align: center;
    padding: 50px 0;
    color: #909399;
}
</style>