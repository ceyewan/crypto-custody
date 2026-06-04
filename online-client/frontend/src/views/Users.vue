<template>
    <div class="page users-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">用户管理</h2>
                <p class="page-subtitle">管理在线系统的管理员、警员、审计员；停用用户后将无法继续登录。</p>
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
                <el-table-column label="状态" width="120">
                    <template slot-scope="scope">
                        <el-tag :type="statusTag(scope.row.status)">{{ statusText(scope.row.status) }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="420">
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
                            style="margin-left: 10px"
                            @click="updateUserRole(scope.row)">
                            更新角色
                        </el-button>
                        <el-button
                            :type="scope.row.status === 'disabled' ? 'success' : 'warning'"
                            size="small"
                            style="margin-left: 10px"
                            @click="toggleUserStatus(scope.row)">
                            {{ scope.row.status === 'disabled' ? '启用用户' : '停用用户' }}
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
  data () {
    return {
      userList: [],
      loading: false
    }
  },
  created () {
    this.fetchUserList()
  },
  methods: {
    async fetchUserList () {
      this.loading = true
      try {
        const response = await userApi.getUsers()
        if (response.data.code !== 200) {
          throw new Error(response.data.message || '获取用户列表失败')
        }
        this.userList = (response.data.data || []).map(user => ({
          ...user,
          newRole: user.role
        }))
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || '获取用户列表失败'
        this.$message.error(errorMsg)
      } finally {
        this.loading = false
      }
    },

    async updateUserRole (user) {
      try {
        const response = await userApi.updateUserRole(user.id, user.newRole)
        if (response.data.code !== 200) {
          throw new Error(response.data.message || '更新角色失败')
        }
        user.role = user.newRole
        this.$message.success(`用户 ${user.nickname || user.username} 角色已更新`)
      } catch (error) {
        user.newRole = user.role
        const errorMsg = error.response?.data?.message || error.message || '更新角色失败'
        this.$message.error(errorMsg)
      }
    },

    async toggleUserStatus (user) {
      const nextStatus = user.status === 'disabled' ? 'active' : 'disabled'
      const actionText = nextStatus === 'active' ? '启用' : '停用'
      try {
        const response = await userApi.updateUserStatus(user.id, nextStatus)
        if (response.data.code !== 200) {
          throw new Error(response.data.message || `${actionText}用户失败`)
        }
        user.status = nextStatus
        this.$message.success(`已${actionText}用户 ${user.nickname || user.username}`)
      } catch (error) {
        const errorMsg = error.response?.data?.message || error.message || `${actionText}用户失败`
        this.$message.error(errorMsg)
      }
    },

    roleText (role) {
      return { admin: '管理员', officer: '警员', auditor: '审计员' }[role] || role
    },

    roleTag (role) {
      if (role === 'admin') return 'danger'
      if (role === 'auditor') return 'info'
      return 'success'
    },

    statusText (status) {
      return status === 'disabled' ? '已停用' : '正常'
    },

    statusTag (status) {
      return status === 'disabled' ? 'warning' : 'success'
    }
  }
}
</script>
