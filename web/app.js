class AccountManager {
    constructor() {
        this.init();
    }

    init() {
        this.loadTokensFromStorage();
        this.updateUI();
        this.bindEvents();
    }

    bindEvents() {
        document.getElementById('registerForm').addEventListener('submit', (e) => {
            e.preventDefault();
            this.register();
        });

        document.getElementById('loginForm').addEventListener('submit', (e) => {
            e.preventDefault();
            this.login();
        });

        document.getElementById('refreshBtn').addEventListener('click', () => {
            this.refresh();
        });

        document.getElementById('logoutBtn').addEventListener('click', () => {
            this.logout();
        });
    }

    async register() {
        const email = document.getElementById('registerEmail').value;
        const password = document.getElementById('registerPassword').value;

        try {
            const response = await fetch('/api/v1/accounts/register', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ email, password }),
            });

            const data = await response.json();
            
            if (response.ok) {
                this.displayResponse(data, true);
                document.getElementById('registerForm').reset();
            } else {
                this.displayResponse(data, false);
            }
        } catch (error) {
            this.displayResponse({ error: error.message }, false);
        }
    }

    async login() {
        const email = document.getElementById('loginEmail').value;
        const password = document.getElementById('loginPassword').value;

        try {
            const response = await fetch('/api/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ email, password }),
            });

            const data = await response.json();
            
            if (response.ok) {
                this.setTokens(data.access_token, data.refresh_token, data.account_id);
                this.displayResponse(data, true);
                document.getElementById('loginForm').reset();
            } else {
                this.displayResponse(data, false);
            }
        } catch (error) {
            this.displayResponse({ error: error.message }, false);
        }
    }

    async refresh() {
        const refreshToken = this.getRefreshToken();
        
        if (!refreshToken) {
            this.displayResponse({ error: 'No refresh token available' }, false);
            return;
        }

        try {
            const response = await fetch('/api/refresh', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ refresh_token: refreshToken }),
            });

            const data = await response.json();
            
            if (response.ok) {
                this.setTokens(data.access_token, data.refresh_token, data.account_id);
                this.displayResponse(data, true);
            } else {
                this.displayResponse(data, false);
                if (response.status === 401) {
                    this.clearTokens();
                }
            }
        } catch (error) {
            this.displayResponse({ error: error.message }, false);
        }
    }

    async logout() {
        const refreshToken = this.getRefreshToken();
        
        if (!refreshToken) {
            this.clearTokens();
            this.displayResponse({ message: 'Already logged out' }, true);
            return;
        }

        try {
            const response = await fetch('/api/logout', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ refresh_token: refreshToken }),
            });

            const data = await response.json();
            this.clearTokens();
            this.displayResponse(data, response.ok);
        } catch (error) {
            this.clearTokens();
            this.displayResponse({ error: error.message }, false);
        }
    }

    setTokens(accessToken, refreshToken, accountId) {
        localStorage.setItem('accessToken', accessToken);
        localStorage.setItem('refreshToken', refreshToken);
        localStorage.setItem('accountId', accountId);
        this.updateUI();
    }

    clearTokens() {
        localStorage.removeItem('accessToken');
        localStorage.removeItem('refreshToken');
        localStorage.removeItem('accountId');
        this.updateUI();
    }

    getAccessToken() {
        return localStorage.getItem('accessToken');
    }

    getRefreshToken() {
        return localStorage.getItem('refreshToken');
    }

    getAccountId() {
        return localStorage.getItem('accountId');
    }

    loadTokensFromStorage() {
        // Tokens are loaded automatically via getters
    }

    updateUI() {
        const statusEl = document.getElementById('status');
        const tokensEl = document.getElementById('tokens');
        const accessTokenEl = document.getElementById('accessToken');
        const refreshTokenEl = document.getElementById('refreshToken');
        const accountIdEl = document.getElementById('accountId');

        const accessToken = this.getAccessToken();
        const refreshToken = this.getRefreshToken();
        const accountId = this.getAccountId();

        if (accessToken && refreshToken) {
            statusEl.textContent = 'Logged in';
            statusEl.className = 'status logged-in';
            tokensEl.classList.remove('hidden');
            
            accessTokenEl.value = accessToken;
            refreshTokenEl.value = refreshToken;
            accountIdEl.textContent = accountId || 'Unknown';
        } else {
            statusEl.textContent = 'Not logged in';
            statusEl.className = 'status logged-out';
            tokensEl.classList.add('hidden');
            
            accessTokenEl.value = '';
            refreshTokenEl.value = '';
            accountIdEl.textContent = '';
        }
    }

    displayResponse(data, isSuccess) {
        const responseEl = document.getElementById('response');
        responseEl.innerHTML = JSON.stringify(data, null, 2);
        responseEl.className = `response-box ${isSuccess ? 'success' : 'error'}`;
    }
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new AccountManager();
});