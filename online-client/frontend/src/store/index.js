import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex)

export default new Vuex.Store({
    state: {
        token: localStorage.getItem('token') || '',
        user: JSON.parse(localStorage.getItem('user')) || null,
        accounts: [],
        transactions: []
    },
    getters: {
        isLoggedIn: state => !!state.token,
        currentUser: state => state.user,
        userRole: state => state.user ? state.user.role : '',
        isAdmin: state => state.user && state.user.role === 'admin',
        isOfficer: state => state.user && (state.user.role === 'officer' || state.user.role === 'admin'),
        isGuest: state => state.user && state.user.role === 'guest',
        userAccounts: state => state.accounts,
        userTransactions: state => state.transactions
    },
    mutations: {
        setToken(state, token) {
            state.token = token
            localStorage.setItem('token', token)
        },
        setUser(state, user) {
            state.user = user
            localStorage.setItem('user', JSON.stringify(user))
        },
        clearToken(state) {
            state.token = ''
            state.user = null
            localStorage.removeItem('token')
            localStorage.removeItem('user')
        },
        setAccounts(state, accounts) {
            state.accounts = accounts
        },
        addAccount(state, account) {
            state.accounts.push(account)
        },
        updateAccount(state, updatedAccount) {
            const index = state.accounts.findIndex(account => account.address === updatedAccount.address)
            if (index !== -1) {
                Vue.set(state.accounts, index, updatedAccount)
            }
        },
        removeAccount(state, address) {
            state.accounts = state.accounts.filter(account => account.address !== address)
        },
        setTransactions(state, transactions) {
            state.transactions = transactions
        },
        addTransaction(state, transaction) {
            state.transactions.unshift(transaction)
        }
    },
    actions: {
        login({ commit }, userData) {
            commit('setToken', userData.token)
            commit('setUser', userData.user)
        },
        logout({ commit }) {
            commit('clearToken')
            commit('setAccounts', [])
            commit('setTransactions', [])
        },
        updateUserInfo({ commit }, user) {
            commit('setUser', user)
        },
        loadAccounts({ commit }, accounts) {
            commit('setAccounts', accounts)
        },
        createAccount({ commit }, account) {
            commit('addAccount', account)
        },
        updateAccountInfo({ commit }, account) {
            commit('updateAccount', account)
        },
        deleteAccount({ commit }, address) {
            commit('removeAccount', address)
        },
        loadTransactions({ commit }, transactions) {
            commit('setTransactions', transactions)
        },
        addNewTransaction({ commit }, transaction) {
            commit('addTransaction', transaction)
        }
    }
})