<template>
    <div class="import-se-container">
        <el-card>
            <div slot="header">
                <span>导入安全芯片</span>
            </div>

            <el-form @submit.native.prevent="handleImport" label-width="120px">
                <el-form-item label="安全芯片ID (SEID)" required>
                    <el-input v-model="seid" placeholder="请输入芯片上贴着的名称 (例如 SExxx)"></el-input>
                </el-form-item>

                <el-form-item>
                    <el-button type="primary" @click="handleImport" :loading="loading">
                        {{ loading ? '导入中...' : '获取CPLC并导入' }}
                    </el-button>
                </el-form-item>
            </el-form>

            <div v-if="cplc" class="result-display">
                <h4>获取到的 CPLC:</h4>
                <pre>{{ cplc }}</pre>
            </div>

        </el-card>
    </div>
</template>

<script>
import { seApi as wailsSeApi } from '../services/wails-api'
import { seApi as cloudSeApi } from '../services/api'

export default {
    name: 'ImportSE',
    data() {
        return {
            seid: '',
            cplc: '',
            loading: false
        }
    },
    methods: {
        async handleImport() {
            // 验证输入
            if (!this.seid || this.seid.trim().length === 0) {
                this.$message.error('请输入有效的安全芯片ID (SEID)')
                return
            }

            // 验证 SEID 格式 (SE + 数字)
            if (!/^SE\d+$/i.test(this.seid.trim())) {
                this.$message.warning('SEID 格式应为 SExxx (SE + 数字)')
            }

            this.loading = true
            this.cplc = ''

            try {
                // 步骤1: 从安全芯片获取 CPLC 数据
                console.log('正在从安全芯片获取 CPLC 数据...')
                const cplcResponse = await wailsSeApi.getCPLC()
                
                if (!cplcResponse.data) {
                    throw new Error('未能从安全芯片获取到 CPLC 数据')
                }

                this.cplc = cplcResponse.data
                console.log('成功获取 CPLC:', this.cplc)

                // 步骤2: 调用云端后端创建安全芯片记录
                console.log('正在调用后端创建安全芯片记录...')
                await cloudSeApi.createSecurityElement(this.seid.trim(), this.cplc)

                // 成功
                this.$message.success(`安全芯片 ${this.seid} 导入成功！`)
                
                // 重置表单
                this.seid = ''
                this.cplc = ''

            } catch (error) {
                console.error('导入安全芯片失败:', error)
                
                // 根据错误类型提供不同的错误信息
                let errorMessage = '导入失败: '
                
                if (error.response) {
                    // HTTP 错误响应
                    errorMessage += error.response.data?.error || error.response.data?.message || '服务器错误'
                    
                    if (error.response.status === 401) {
                        errorMessage = '认证失败，请重新登录'
                    } else if (error.response.status === 403) {
                        errorMessage = '权限不足，请联系管理员'
                    } else if (error.response.status >= 500) {
                        errorMessage = '服务器内部错误，请稍后重试'
                    }
                } else if (error.message) {
                    // Wails 或其他错误
                    errorMessage += error.message
                } else {
                    errorMessage += '未知错误'
                }
                
                this.$message.error(errorMessage)
            } finally {
                this.loading = false
            }
        }
    }
}
</script>

<style scoped>
.import-se-container {
    padding: 20px;
}

.result-display {
    margin-top: 20px;
    padding: 15px;
    background-color: #f5f7fa;
    border: 1px solid #e4e7ed;
    border-radius: 4px;
}

pre {
    white-space: pre-wrap;
    word-wrap: break-word;
    color: #606266;
}
</style>