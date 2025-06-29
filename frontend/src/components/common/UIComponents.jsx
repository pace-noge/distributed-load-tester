import React from 'react';
import { CheckCircle, XCircle, AlertCircle, Clock } from 'lucide-react';

export const LoadingSpinner = ({ size = "default" }) => {
    const sizeClasses = {
        small: "h-4 w-4",
        default: "h-8 w-8",
        large: "h-12 w-12"
    };

    return (
        <div className={`animate-spin rounded-full ${sizeClasses[size]} border-b-2 border-blue-600`}></div>
    );
};

export const StatusBadge = ({ status, className = "" }) => {
    const getStatusConfig = (status) => {
        switch (status?.toUpperCase()) {
            case 'COMPLETED':
                return { bg: 'bg-green-100', text: 'text-green-800', icon: CheckCircle };
            case 'RUNNING':
                return { bg: 'bg-blue-100', text: 'text-blue-800', icon: Clock };
            case 'FAILED':
                return { bg: 'bg-red-100', text: 'text-red-800', icon: XCircle };
            case 'PENDING':
                return { bg: 'bg-yellow-100', text: 'text-yellow-800', icon: AlertCircle };
            default:
                return { bg: 'bg-gray-100', text: 'text-gray-800', icon: AlertCircle };
        }
    };

    const config = getStatusConfig(status);
    const Icon = config.icon;

    return (
        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${config.bg} ${config.text} ${className}`}>
            <Icon className="w-3 h-3 mr-1" />
            {status}
        </span>
    );
};
