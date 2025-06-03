<template>
    <div class="users-container">
        <el-card>
            <div slot="header">
                <span>用户管理</span>
            </div>

            <el-table :data="userList" v-loading="loading" style="width: 100%">
                <el-table-column prop="username" label="用户名" width="180"></el-table-column>
                <el-table-column prop="email" label="邮箱" width="220"></el-table-column>
                <el-table-column prop="role" label="当前角色" width="120"></el-table-column>
                <el-table-column label="操作">
                    <template slot-scope="scope">
                        <el-select v-model="scope.row.newRole" placeholder="选择角色" size="small">
                            <el-option label="管理员" value="admin"></el-option>
                            <el-option label="协调者" value="coordinator"></el-option>
                            <el-option label="参与者" value="participant"></el-option>
                            <el-option label="访客" value="guest"></el-option>
                        </el-select>
                        <el-button type="primary" size="small" :disabled="scope.row.role === scope.row.newRole"
                            @click="updateUserRole(scope.row)" style="margin-left: 10px">
                            更新角色
                        </el-button>
                    </template>
                </el-table-column>
            </el-table>

            <div v-if="userList.length === 0 && !loading" class="empty-state">
                <p>暂无用户数据</p>
                <el-button type="primary" size="small" @click="fetchUserList">刷新</el-button>
            </div>
        </el-card>
    </div>
</template>

<script>
import { userApi } from '../services/api'

export default {
    name: 'Users',
    data() {
        return {
            userList: [],
            loading: false
        }
    },
    created() {
        this.fetchUserList()
    },
    methods: {
        async fetchUserList() {
            this.loading = true
            try {
                const response = await userApi.getUsers()
                // 检查响应格式，适配后端API的返回格式
                const users = response.data.users || response.data.data || []
                this.userList = users.map(user => ({
                    ...user,
                    newRole: user.role
                }))

                if (this.userList.length === 0) {
                    this.$message.warning('未找到用户数据')
                }
            } catch (error) {
                console.error('获取用户列表失败:', error)
                this.$message.error('获取用户列表失败: ' + (error.response?.data?.error || error.message))
            } finally {
                this.loading = false
            }
        },

        async updateUserRole(user) {
            try {
                await userApi.updateUserRole(user.username, user.newRole)
                this.$message.success(`用户 ${user.username} 角色已更新为 ${user.newRole}`)
                // 更新当前角色
                user.role = user.newRole
            } catch (error) {
                console.error('更新用户角色失败:', error)
                this.$message.error('更新用户角色失败')
            }
        }
    }
}
</script>

<style scoped>
.users-container {
    padding: 20px;
}

.empty-state {
    text-align: center;
    padding: 40px 0;
    color: #606266;
}
</style>