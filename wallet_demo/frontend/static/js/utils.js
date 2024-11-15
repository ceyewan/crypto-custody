class Utils {
    static async fetchJSON(url, options = {}) {
        options.headers = {
            'Content-Type': 'application/json',
            ...options.headers
        };

        const response = await fetch(url, options);
        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || '请求失败');
        }

        return data;
    }

    static updateAccountsList(accounts) {
        const accountsList = document.getElementById('accounts-list');
        accountsList.innerHTML = accounts.map(account => `
            <li>
                <div>地址: ${account.Address}</div>
                <div>余额: ${account.Balance}</div>
            </li>
        `).join('');
    }

    static showError(message, error = null) {
        alert(`错误: ${message}\n${error ? error.message : ''}`);
    }

    static showSuccess(message) {
        alert(message);
    }
}

// 页面加载完成后自动获取账户列表
document.addEventListener('DOMContentLoaded', () => AccountAPI.getAccounts());