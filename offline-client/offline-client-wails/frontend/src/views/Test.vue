<template>
    <div class="test-container">
        <el-card>
            <div slot="header">
                <span>Wails MPC 功能测试</span>
            </div>

            <el-row :gutter="20">
                <el-col :span="12">
                    <el-card shadow="hover">
                        <div slot="header">
                            <span>本地 MPC 测试</span>
                        </div>
                        
                        <el-button 
                            type="primary" 
                            :loading="testLoading" 
                            @click="testLocalMpc"
                            style="width: 100%; margin-bottom: 10px;">
                            测试本地密钥生成
                        </el-button>
                        
                        <el-button 
                            type="success" 
                            :loading="signLoading" 
                            @click="testLocalSign"
                            style="width: 100%; margin-bottom: 10px;">
                            测试本地签名
                        </el-button>
                        
                        <el-button 
                            type="info" 
                            :loading="cplcLoading" 
                            @click="testLocalCPLC"
                            style="width: 100%;">
                            测试获取CPLC
                        </el-button>
                    </el-card>
                </el-col>

                <el-col :span="12">
                    <el-card shadow="hover">
                        <div slot="header">
                            <span>测试结果</span>
                        </div>
                        
                        <div v-if="testResults.length === 0" style="text-align: center; color: #909399;">
                            暂无测试结果
                        </div>
                        
                        <div v-for="(result, index) in testResults" :key="index" style="margin-bottom: 15px;">
                            <el-tag :type="result.success ? 'success' : 'danger'" style="margin-bottom: 5px;">
                                {{ result.operation }} - {{ result.success ? '成功' : '失败' }}
                            </el-tag>
                            <div style="font-size: 12px; color: #666;">
                                {{ result.message }}
                            </div>
                            <div v-if="result.data" style="margin-top: 5px;">
                                <el-input 
                                    :value="JSON.stringify(result.data, null, 2)" 
                                    type="textarea" 
                                    :rows="3" 
                                    readonly>
                                </el-input>
                            </div>
                        </div>
                    </el-card>
                </el-col>
            </el-row>
        </el-card>
    </div>
</template>

<script>
export default {
    name: 'Test',
    data() {
        return {
            testLoading: false,
            signLoading: false,
            cplcLoading: false,
            testResults: []
        }
    },
    methods: {
        async testLocalMpc() {
            this.testLoading = true
            
            try {
                console.log('开始测试本地密钥生成...')
                const result = await this.$localMpcApi.keyGen({
                    threshold: 2,
                    total_parts: 3,
                    participants: ['user1', 'user2', 'user3']
                })
                
                this.addTestResult('密钥生成', true, '密钥生成完成', result.data)
                this.$message.success('本地密钥生成测试成功！')
            } catch (error) {
                console.error('本地密钥生成测试失败:', error)
                this.addTestResult('密钥生成', false, error.message)
                this.$message.error('本地密钥生成测试失败: ' + error.message)
            } finally {
                this.testLoading = false
            }
        },
        
        async testLocalSign() {
            this.signLoading = true
            
            try {
                console.log('开始测试本地签名...')
                const result = await this.$localMpcApi.sign({
                    message: 'Hello Wails MPC Test!',
                    data: 'Test message for signing',
                    threshold: 2,
                    total_parts: 3
                })
                
                this.addTestResult('消息签名', true, '签名完成', result.data)
                this.$message.success('本地签名测试成功！')
            } catch (error) {
                console.error('本地签名测试失败:', error)
                this.addTestResult('消息签名', false, error.message)
                this.$message.error('本地签名测试失败: ' + error.message)
            } finally {
                this.signLoading = false
            }
        },
        
        async testLocalCPLC() {
            this.cplcLoading = true
            
            try {
                console.log('开始测试获取CPLC...')
                const result = await this.$localSeApi.getCPLC()
                
                this.addTestResult('获取CPLC', true, 'CPLC信息获取完成', result.data)
                this.$message.success('获取CPLC测试成功！')
            } catch (error) {
                console.error('获取CPLC测试失败:', error)
                this.addTestResult('获取CPLC', false, error.message)
                this.$message.error('获取CPLC测试失败: ' + error.message)
            } finally {
                this.cplcLoading = false
            }
        },
        
        addTestResult(operation, success, message, data = null) {
            this.testResults.unshift({
                operation,
                success,
                message,
                data,
                timestamp: new Date().toLocaleString()
            })
            
            // 保持最多10条记录
            if (this.testResults.length > 10) {
                this.testResults = this.testResults.slice(0, 10)
            }
        }
    }
}
</script>

<style scoped>
.test-container {
    padding: 20px;
}
</style>
