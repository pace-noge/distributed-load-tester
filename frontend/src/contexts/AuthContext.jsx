import { useState, createContext, useContext, useEffect } from 'react';
import { getUserProfile } from '../utils/api.js';

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
    const [token, setToken] = useState(localStorage.getItem('jwt_token'));
    const [isLoggedIn, setIsLoggedIn] = useState(!!token);
    const [user, setUser] = useState(null);

    // Load user profile when token exists
    useEffect(() => {
        const loadUserProfile = async () => {
            if (token) {
                try {
                    const profile = await getUserProfile();
                    setUser(profile);
                } catch (error) {
                    console.error('Failed to load user profile:', error);
                    // If token is invalid, logout
                    logout();
                }
            }
        };

        if (token && !user) {
            loadUserProfile();
        }
    }, [token, user]);

    const login = async (jwtToken) => {
        localStorage.setItem('jwt_token', jwtToken);
        setToken(jwtToken);
        setIsLoggedIn(true);

        // Load user profile after login
        try {
            const profile = await getUserProfile();
            setUser(profile);
        } catch (error) {
            console.error('Failed to load user profile after login:', error);
        }
    };

    const logout = () => {
        localStorage.removeItem('jwt_token');
        localStorage.removeItem('user_profile');
        setToken(null);
        setUser(null);
        setIsLoggedIn(false);
    };

    const getToken = () => token;

    return (
        <AuthContext.Provider value={{ isLoggedIn, user, login, logout, getToken }}>
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = () => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
};
