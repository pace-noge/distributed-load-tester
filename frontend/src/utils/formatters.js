/**
 * Formats a date string for display
 * @param {string} dateString - The date string to format
 * @returns {string} - Formatted date string or fallback
 */
export const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    try {
        const date = new Date(dateString);
        if (isNaN(date.getTime())) return 'Invalid Date';
        return date.toLocaleString();
    } catch (error) {
        return 'Invalid Date';
    }
};

/**
 * Formats a duration for display
 * @param {string|number} duration - The duration to format
 * @returns {string} - Formatted duration string
 */
export const formatDuration = (duration) => {
    if (!duration) return 'N/A';
    // Handle both string format like "10s" and numeric formats
    if (typeof duration === 'string') return duration;
    if (typeof duration === 'number') return duration + 's';
    return duration.toString();
};

/**
 * Formats rate distribution for display
 * @param {string} distribution - The rate distribution to format
 * @returns {string} - Formatted distribution string
 */
export const formatRateDistribution = (distribution) => {
    if (!distribution) return 'uniform';
    return distribution.toLowerCase();
};

/**
 * Gets the display value with fallbacks
 * @param {object} obj - The object to get value from
 * @param {string[]} keys - Array of keys to try in order
 * @param {any} defaultValue - Default value if none found
 * @returns {any} - The value or default
 */
export const getValueWithFallback = (obj, keys, defaultValue = 'N/A') => {
    for (const key of keys) {
        if (obj && obj[key] !== undefined && obj[key] !== null) {
            return obj[key];
        }
    }
    return defaultValue;
};

/**
 * Gets page numbers for pagination
 * @param {number} currentPage - Current page number
 * @param {number} totalPages - Total number of pages
 * @returns {Array} - Array of page numbers and ellipsis
 */
export const getPageNumbers = (currentPage, totalPages) => {
    const pages = [];
    const showEllipsis = totalPages > 7;

    if (!showEllipsis) {
        for (let i = 1; i <= totalPages; i++) {
            pages.push(i);
        }
    } else {
        if (currentPage <= 4) {
            for (let i = 1; i <= 5; i++) pages.push(i);
            pages.push('...');
            pages.push(totalPages);
        } else if (currentPage >= totalPages - 3) {
            pages.push(1);
            pages.push('...');
            for (let i = totalPages - 4; i <= totalPages; i++) pages.push(i);
        } else {
            pages.push(1);
            pages.push('...');
            for (let i = currentPage - 1; i <= currentPage + 1; i++) pages.push(i);
            pages.push('...');
            pages.push(totalPages);
        }
    }

    return pages;
};
