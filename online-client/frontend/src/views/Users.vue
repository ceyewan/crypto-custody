<template>
    <div class="users-container">
        <el-card>
            <div slot="header" class="clearfix">
                <span>用户管理</span>
                <el-button style="float: right; padding: 3px 0" type="text" @click="refreshUserList">
                    刷新
                </el-button>
            </div>

            <el-table :data="userList" v-loading="loading" style="width: 100%">
                <el-table-column prop="id" label="ID" width="80"></el-table-column>
                <el-table-column prop="username" label="用户名" width="150"></el-table-column>
                <el-table-column prop="email" label="邮箱" width="200"></el-table-column>
                <el-table-column prop="role" label="当前角色" width="120">
                    <template slot-scope="scope">
                        <el-tag :type="getRoleTagType(scope.row.role)">{{ getRoleText(scope.row.role) }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column label="操作">
                    <template slot-scope="scope">
                        <el-select v-model="scope.row.newRole" placeholder="选择角色" size="small" style="width: 120px">
                            <el-option label="管理员" value="admin"></el-option>
                            <el-option label="警员" value="officer"></el-option>
                            <el-option label="访客" value="guest"></el-option>
                        </el-select>
                        <el-button type="primary" size="small" :disabled="scope.row.role === scope.row.newRole"
                            @click="updateUserRole(scope.row)" style="margin-left: 10px">
                            更新角色
                        </el-button>
                        <el-button type="warning" size="small" @click="showEditUserDialog(scope.row)" style="margin-left: 5px">
                            编辑
                        </el-button>
                        <el-button type="danger" size="small" @click="deleteUser(scope.row)" style="margin-left: 5px">
                            删除
                        </el-button>
                    </template>
                </el-table-column>
            </el-table>

            <div v-if="userList.length === 0 && !loading" class="empty-state">
                <p>暂无用户数据</p>
                <el-button type="primary" size="small" @click="refreshUserList">刷新</el-button>
            </div>
        </el-card>

        <!-- 编辑用户对话框 -->
        <el-dialog title="编辑用户" :visible.sync="editDialogVisible" width="400px">
            <el-form :model="editForm" :rules="editRules" ref="editForm" label-width="80px">
                <el-form-item label="用户名" prop="username">
                    <el-input v-model="editForm.username"></el-input>
                </el-form-item>
                <el-form-item label="新密码" prop="newPassword">
                    <el-input v-model="editForm.newPassword" type="password" placeholder="留空则不修改密码"></el-input>
                </el-form-item>
            </el-form>
            <div slot="footer" class="dialog-footer">
                <el-button @click="editDialogVisible = false">取 消</el-button>
                <el-button type="primary" @click="handleEditUser" :loading="editLoading">确 定</el-button>
            </div>
        </el-dialog>
    </div>
</template>

<script>
import { userApi } from '../services/api'

export default {
  name: 'Users',
  data () {
    return {
      userList: [],
      loading: false,
      editDialogVisible: false,
      editLoading: false,
      editForm: {
        id: null,
        username: '',
        newPassword: ''
      },
      editRules: {
        username: [
          { required: true, message: '请输入用户名', trigger: 'blur' },
          { min: 3, max: 20, message: '用户名长度应为3-20个字符', trigger: 'blur' }
        ],
        newPassword: [
          { min: 6, message: '密码长度至少为6个字符', trigger: 'blur' }
        ]
      }
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
        if (response.data.code === 200) {
          this.userList = response.data.data.map(user => ({
            ...user,
            newRole: user.role
          }))
        } else {
          throw new Error(response.data.message || '获取用户列表失败')
        }

        if (this.userList.length === 0) {
          this.$message.warning('未找到用户数据')
        }
      } catch (error) {
        console.error('获取用户列表失败:', error)
        let errorMsg = '获取用户列表失败'
        if (error.response && error.response.data) {
          errorMsg = error.response.data.message || errorMsg
        } else if (error.message) {
          errorMsg = error.message
        }
        this.$message.error(errorMsg)
      } finally {
        this.loading = false
      }
    },

    async updateUserRole (user) {
      try {
        const response = await userApi.updateUserRole(user.id, user.newRole)
        if (response.data.code === 200) {
          this.$message.success(`用户 ${user.username} 角色已更新为 ${this.getRoleText(user.newRole)}`)
          // 更新当前角色
          user.role = user.newRole
        } else {
          throw new Error(response.data.message || '更新角色失败')
        }
      } catch (error) {
        console.error('更新用户角色失败:', error)
        let errorMsg = '更新用户角色失败'
        if (error.response && error.response.data) {
          errorMsg = error.response.data.message || errorMsg
        }
        this.$message.error(errorMsg)
        // 恢复原来的值
        user.newRole = user.role
      }
    },

    showEditUserDialog (user) {
      this.editForm = {
        id: user.id,
        username: user.username,
        newPassword: ''
      }
      this.editDialogVisible = true
    },

    async handleEditUser () {
      this.$refs.editForm.validate(async valid => {
        if (!valid) {
          return false
        }

        this.editLoading = true

        try {
          // 更新用户名
          const usernameResponse = await userApi.updateUsername(this.editForm.id, this.editForm.username)
          if (usernameResponse.data.code !== 200) {
            throw new Error(usernameResponse.data.message || '更新用户名失败')
          }

          // 如果有新密码则更新密码
          if (this.editForm.newPassword) {
            const passwordResponse = await userApi.adminUpdatePassword(this.editForm.id, this.editForm.newPassword)
            if (passwordResponse.data.code !== 200) {
              throw new Error(passwordResponse.data.message || '更新密码失败')
            }
          }

          this.$message.success('用户信息更新成功')
          this.editDialogVisible = false
          this.fetchUserList() // 刷新列表
        } catch (error) {
          console.error('更新用户信息失败:', error)
          let errorMsg = '更新用户信息失败'
          if (error.response && error.response.data) {
            errorMsg = error.response.data.message || errorMsg
          } else if (error.message) {
            errorMsg = error.message
          }
          this.$message.error(errorMsg)
        } finally {
          this.editLoading = false
        }
      })
    },

    async deleteUser (user) {
      try {
        const confirm = await this.$confirm(
                    `确定要删除用户 "${user.username}" 吗？此操作不可撤销。`,
                    '确认删除',
                    {
                      confirmButtonText: '确定',
                      cancelButtonText: '取消',
                      type: 'warning'
                    }
        )

        if (confirm) {
          const response = await userApi.deleteUser(user.id)
          if (response.data.code === 200) {
            this.$message.success('用户删除成功')
            this.fetchUserList() // 刷新列表
          } else {
            throw new Error(response.data.message || '删除用户失败')
          }
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('删除用户失败:', error)
          let errorMsg = '删除用户失败'
          if (error.response && error.response.data) {
            errorMsg = error.response.data.message || errorMsg
          } else if (error.message) {
            errorMsg = error.message
          }
          this.$message.error(errorMsg)
        }
      }
    },

    refreshUserList () {
      this.fetchUserList()
    },

    getRoleText (role) {
      const roleMap = {
        admin: '管理员',
        officer: '警员',
        guest: '访客'
      }
      return roleMap[role] || role
    },

    getRoleTagType (role) {
      const typeMap = {
        admin: 'danger',
        officer: 'warning',
        guest: 'info'
      }
      return typeMap[role] || 'info'
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

.dialog-footer {
    text-align: right;
}
</style>
