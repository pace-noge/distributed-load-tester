import { useState, createContext, useContext } from 'react';

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
    const [token, setToken] = useState(localStorage.getItem('jwt_token'));
    const [isLoggedIn, setIsLoggedIn] = useState(!!token);

    const login = (jwtToken) => {
        localStorage.setItem('jwt_token', jwtToken);
        setToken(jwtToken);
        setIsLoggedIn(true);
    };

    const logout = () => {
        localStorage.removeItem('jwt_token');
        setToken(null);
        setIsLoggedIn(false);
    };

    const getToken = () => token;

    return (
        <AuthContext.Provider value={{ isLoggedIn, login, logout, getToken }}>
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
