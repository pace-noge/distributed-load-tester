import { API_BASE_URL } from './constants.js';

/**
 * Authenticated fetch wrapper
 * @param {string} url - The URL to fetch
 * @param {object} options - Fetch options
 * @returns {Promise} - The response data
 */
export const authenticatedFetch = async (url, options = {}) => {
    const token = localStorage.getItem('jwt_token');
    if (!token) {
        throw new Error('Not authenticated');
    }

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        ...options.headers,
    };

    const response = await fetch(url, { ...options, headers });
    if (!response.ok) {
        if (response.status === 401 || response.status === 403) {
            const event = new CustomEvent('auth-error');
            window.dispatchEvent(event);
        }
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `API Error: ${response.status}`);
    }
    return response.json();
};

/**
 * Fetch test history data with pagination
 * @param {number} page - Page number (1-based)
 * @param {number} limit - Number of items per page
 * @returns {Promise} - Paginated test history data
 */
export const fetchTestHistory = async (page = 1, limit = 20) => {
    const offset = (page - 1) * limit;
    const url = `${API_BASE_URL}/tests?limit=${limit}&offset=${offset}`;
    const data = await authenticatedFetch(url);
    return data;
};

/**
 * Fetch all test history data (backwards compatibility)
 * @returns {Promise} - All test history data
 */
export const fetchAllTestHistory = async () => {
    // For backwards compatibility, fetch all tests by using a large page size
    const data = await fetchTestHistory(1, 1000);
    return Array.isArray(data) ? data : (data.tests || data.data || []);
};

/**
 * Fetch test detail data
 * @param {string} testId - Test ID
 * @returns {Promise} - Combined test detail data
 */
export const fetchTestDetail = async (testId) => {
    // First, get all tests to find the specific test details
    const testsData = await fetchAllTestHistory();
    const tests = Array.isArray(testsData) ? testsData : (testsData.tests || testsData.data || []);
    const testInfo = tests.find(test => test.id === testId);

    if (!testInfo) {
        throw new Error(`Test with ID ${testId} not found`);
    }

    // Fetch test results and aggregated results in parallel
    const [resultsData, aggregatedData] = await Promise.all([
        authenticatedFetch(`${API_BASE_URL}/tests/${testId}/results`)
            .catch(_err => {
                // Failed to fetch test results - this is expected for some tests
                return null;
            }),
        authenticatedFetch(`${API_BASE_URL}/tests/${testId}/aggregated-result`)
            .catch(_err => {
                // Failed to fetch aggregated results - this is expected for some tests
                return null;
            })
    ]);

    // Combine all the data
    return {
        test: testInfo,
        results: resultsData || [],
        aggregated_result: aggregatedData || null
    };
};

/**
 * Replay a test
 * @param {string} testId - Test ID
 * @param {string} testName - Test name
 * @returns {Promise} - Replay response
 */
export const replayTest = async (testId, testName) => {
    // First, get the test details to extract the original configuration
    const testDetail = await fetchTestDetail(testId);
    const originalTest = testDetail.test;

    if (!originalTest) {
        throw new Error('Original test configuration not found');
    }

    // Create a new test request with the same configuration but new name
    // Use snake_case field names to match the backend expectations
    const replayRequest = {
        name: `${testName} (Replay)`,
        vegeta_payload_json: originalTest.vegetaPayloadJson || originalTest.vegetaPayloadJSON,
        duration_seconds: originalTest.durationSeconds?.toString() || originalTest.duration_seconds?.toString(),
        rate_per_second: originalTest.ratePerSecond || originalTest.rate_per_second,
        targets_base64: originalTest.targetsBase64 || originalTest.targets_base64,
        worker_count: originalTest.workerCount || originalTest.worker_count,
        rate_distribution: originalTest.rateDistribution || originalTest.rate_distribution || "same"
    };

    // Submit the new test using the existing submit endpoint
    return await authenticatedFetch(`${API_BASE_URL}/test/submit`, {
        method: 'POST',
        body: JSON.stringify(replayRequest)
    });
};

// =============================================================================
// USER MANAGEMENT API FUNCTIONS
// =============================================================================

/**
 * Login user
 * @param {string} username - Username
 * @param {string} password - Password
 * @returns {Promise} - Login response with token
 */
export const loginUser = async (username, password) => {
    const response = await fetch(`${API_BASE_URL}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: 'Login failed' }));
        throw new Error(errorData.message || 'Login failed');
    }

    return response.json();
};

/**
 * Get current user profile
 * @returns {Promise} - User profile data
 */
export const getUserProfile = async () => {
    return await authenticatedFetch(`${API_BASE_URL}/auth/profile`);
};

/**
 * Change user password
 * @param {string} currentPassword - Current password
 * @param {string} newPassword - New password
 * @returns {Promise} - Success response
 */
export const changePassword = async (currentPassword, newPassword) => {
    return await authenticatedFetch(`${API_BASE_URL}/auth/change-password`, {
        method: 'POST',
        body: JSON.stringify({
            currentPassword,
            newPassword
        })
    });
};

/**
 * Get all users (admin only)
 * @returns {Promise} - Array of users
 */
export const getAllUsers = async () => {
    return await authenticatedFetch(`${API_BASE_URL}/users`);
};

/**
 * Create a new user (admin only)
 * @param {object} userData - User data
 * @returns {Promise} - Created user data
 */
export const createUser = async (userData) => {
    return await authenticatedFetch(`${API_BASE_URL}/users/create`, {
        method: 'POST',
        body: JSON.stringify(userData)
    });
};

/**
 * Update user profile
 * @param {object} userData - Updated user data
 * @returns {Promise} - Updated user data
 */
export const updateUserProfile = async (userData) => {
    return await authenticatedFetch(`${API_BASE_URL}/users`, {
        method: 'PUT',
        body: JSON.stringify(userData)
    });
};

/**
 * Get user by ID (admin only)
 * @param {string} userId - User ID
 * @returns {Promise} - User data
 */
export const getUserById = async (userId) => {
    return await authenticatedFetch(`${API_BASE_URL}/users/${userId}`);
};

/**
 * Activate user (admin only)
 * @param {string} userId - User ID
 * @returns {Promise} - Success response
 */
export const activateUser = async (userId) => {
    return await authenticatedFetch(`${API_BASE_URL}/users/${userId}/activate`, {
        method: 'POST'
    });
};

/**
 * Deactivate user (admin only)
 * @param {string} userId - User ID
 * @returns {Promise} - Success response
 */
export const deactivateUser = async (userId) => {
    return await authenticatedFetch(`${API_BASE_URL}/users/${userId}/deactivate`, {
        method: 'POST'
    });
};

/**
 * Logout user (clear local storage)
 */
export const logoutUser = () => {
    localStorage.removeItem('jwt_token');
    localStorage.removeItem('user_profile');
    window.location.href = '/login';
};

/**
 * Get analytics overview
 * @param {object} options - Analytics options
 * @param {string} options.startDate - Start date (YYYY-MM-DD)
 * @param {string} options.endDate - End date (YYYY-MM-DD)
 * @returns {Promise} - Analytics overview data
 */
export const getAnalyticsOverview = async (options = {}) => {
    const params = new URLSearchParams();
    if (options.startDate) params.append('startDate', options.startDate);
    if (options.endDate) params.append('endDate', options.endDate);

    const url = `${API_BASE_URL}/analytics/overview${params.toString() ? '?' + params.toString() : ''}`;
    return await authenticatedFetch(url);
};

/**
 * Get target analytics
 * @param {object} options - Analytics options
 * @param {string} options.startDate - Start date (YYYY-MM-DD)
 * @param {string} options.endDate - End date (YYYY-MM-DD)
 * @param {string} options.target - Target URL filter
 * @returns {Promise} - Target analytics data
 */
export const getTargetAnalytics = async (options = {}) => {
    const params = new URLSearchParams();
    if (options.startDate) params.append('startDate', options.startDate);
    if (options.endDate) params.append('endDate', options.endDate);
    if (options.target) params.append('target', options.target);

    const url = `${API_BASE_URL}/analytics/targets${params.toString() ? '?' + params.toString() : ''}`;
    return await authenticatedFetch(url);
};
