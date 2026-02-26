import { type ThemeConfig, theme } from 'antd';

export const themeConfig: ThemeConfig = {
    algorithm: theme.darkAlgorithm,
    token: {
        colorPrimary: '#60a5fa', // Lighter Blue for better contrast on dark
        colorBgBase: '#0f172a',
        borderRadius: 12, // More rounded, modern look
        fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
        boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.3), 0 2px 4px -1px rgba(0, 0, 0, 0.16)',
    },
    components: {
        Layout: {
            colorBgHeader: 'transparent', // Let glass effect in App.tsx handle it
            colorBgBody: 'transparent', // Transparent to show global gradient
            colorBgTrigger: 'rgba(30, 41, 59, 0.5)',
        },
        Card: {
            colorBgContainer: 'rgba(30, 41, 59, 0.6)', // Fallback if glass class not used
            boxShadowTertiary: '0 1px 3px 0 rgba(0, 0, 0, 0.3), 0 1px 2px 0 rgba(0, 0, 0, 0.2)',
        },
        Button: {
            fontWeight: 500,
            controlHeight: 40, // Slightly taller buttons suitable for "premium" feel
            borderRadius: 8,
        },
        Table: {
            headerBg: 'rgba(30, 41, 59, 0.4)',
            headerColor: '#cbd5e1',
            rowHoverBg: 'rgba(51, 65, 85, 0.4)',
            colorBgContainer: 'rgba(30, 41, 59, 0.4)', // Semi-transparent tables
        },
        Menu: {
            itemBg: 'transparent',
            darkItemBg: 'transparent',
            popupBg: '#1e293b',
        }
    },
};
