<template>
    <div class="page users-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">账户管理</h2>
                <p class="page-subtitle">只保留管理员、警员、审计员三类角色；警员和管理员可以参与 MPC。</p>
            </div>
            <el-button icon="el-icon-refresh" :loading="loading" @click="fetchUserList">刷新</el-button>
        </div>

        <el-card>
            <el-table :data="userList" v-loading="loading" style="width: 100%">
                <el-table-column prop="identifier" label="登录标识" min-width="180">
                    <template slot-scope="scope">
                        {{ scope.row.identifier || scope.row.username }}
                    </template>
                </el-table-column>
                <el-table-column prop="nickname" label="昵称" min-width="160">
                    <template slot-scope="scope">
                        {{ scope.row.nickname || '-' }}
                    </template>
                </el-table-column>
                <el-table-column label="当前角色" width="120">
                    <template slot-scope="scope">
                        <el-tag :type="roleTag(scope.row.role)">{{ roleText(scope.row.role) }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="status" label="状态" width="120"></el-table-column>
                <el-table-column label="操作" width="320">
                    <template slot-scope="scope">
                        <el-select v-model="scope.row.newRole" placeholder="选择角色" size="small">
                            <el-option label="管理员" value="admin"></el-option>
                            <el-option label="警员" value="officer"></el-option>
                            <el-option label="审计员" value="auditor"></el-option>
                        </el-select>
                        <el-button
                            type="primary"
                            size="small"
                            :disabled="scope.row.role === scope.row.newRole"
                            @click="updateUserRole(scope.row)"
                            style="margin-left: 10px">
                            更新角色
                        </el-button>
                    </template>
                </el-table-column>
            </el-table>

            <el-empty v-if="userList.length === 0 && !loading" description="暂无用户数据" :image-size="90"></el-empty>
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
                const users = response.data.users || response.data.data || []
                this.userList = users.map(user => ({
                    ...user,
                    newRole: user.role
                }))
            } catch (error) {
                this.$message.error('获取用户列表失败: ' + (error.response?.data?.error || error.message))
            } finally {
                this.loading = false
            }
        },

        async updateUserRole(user) {
            try {
                await userApi.updateUserRole(user.username || user.identifier, user.newRole)
                this.$message.success(`用户 ${user.nickname || user.username} 角色已更新`)
                user.role = user.newRole
            } catch (error) {
                this.$message.error(error.response?.data?.error || '更新用户角色失败')
            }
        },

        roleText(role) {
            const map = { admin: '管理员', officer: '警员', auditor: '审计员' }
            return map[role] || role
        },

        roleTag(role) {
            if (role === 'admin') return 'danger'
            if (role === 'auditor') return 'info'
            return 'success'
        }
    }
}
</script>
