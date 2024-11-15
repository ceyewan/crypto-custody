const API_BASE_URL = 'http://localhost:8080/api';

class AccountAPI {
    static async getAccounts() {
        try {
            const accounts = await Utils.fetchJSON(`${API_BASE_URL}/accounts/`);
            Utils.updateAccountsList(accounts);
        } catch (error) {
            Utils.showError('获取账户列表失败', error);
        }
    }

    static async createAccount() {
        const publicKeyX = document.getElementById('publicKeyX').value;
        const publicKeyY = document.getElementById('publicKeyY').value;

        if (!publicKeyX || !publicKeyY) {
            Utils.showError('请填写完整的公钥信息');
            return;
        }

        try {
            const response = await Utils.fetchJSON(`${API_BASE_URL}/accounts/`, {
                method: 'POST',
                body: JSON.stringify({ PublicKeyX: publicKeyX, PublicKeyY: publicKeyY })
            });
            Utils.showSuccess('账户创建成功');
            this.getAccounts();
        } catch (error) {
            Utils.showError('创建账户失败', error);
        }
    }

    static async transferAll() {
        try {
            await Utils.fetchJSON(`${API_BASE_URL}/accounts/transferAll`);
            Utils.showSuccess('转账操作已完成');
            this.getAccounts();
        } catch (error) {
            Utils.showError('转账操作失败', error);
        }
    }

    static async updateBalance() {
        try {
            await Utils.fetchJSON(`${API_BASE_URL}/accounts/updateBalance`);
            Utils.showSuccess('余额更新完成');
            this.getAccounts();
        } catch (error) {
            Utils.showError('更新余额失败', error);
        }
    }

    static async packTransferData() {
        const fromAddress = document.getElementById('fromAddress').value;
        const toAddress = document.getElementById('toAddress').value;
        const amount = parseFloat(document.getElementById('amount').value);

        if (!fromAddress || !toAddress || isNaN(amount)) {
            Utils.showError('请填写完整的交易信息');
            return;
        }

        try {
            const response = await Utils.fetchJSON(`${API_BASE_URL}/accounts/packTransferData`, {
                method: 'POST',
                body: JSON.stringify({ from: fromAddress, to: toAddress, amount: amount })
            });
            Utils.showSuccess('交易数据打包成功');
            console.log('Packed Data:', response.data);
            document.getElementById('packedDataOutput').value = response.data;
        } catch (error) {
            Utils.showError('打包交易数据失败', error);
        }
    }

    static async sendTransaction() {
        const signature = document.getElementById('signature').value;

        if (!signature) {
            Utils.showError('请填写签名信息');
            return;
        }

        try {
            await Utils.fetchJSON(`${API_BASE_URL}/accounts/sendTransaction`, {
                method: 'POST',
                body: JSON.stringify({ signature: signature })
            });
            Utils.showSuccess('交易发送成功');
        } catch (error) {
            Utils.showError('发送交易失败', error);
        }
    }
}