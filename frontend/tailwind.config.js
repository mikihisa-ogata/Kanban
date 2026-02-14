/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      maxWidth: {
        sm: '400px',
      },
      minWidth: {
        64: '256px',
      },
      animation: {
        slideIn: 'slideIn 0.3s ease forwards',
      },
      keyframes: {
        slideIn: {
          from: {
            transform: 'translateX(100%)',
            opacity: '0',
          },
          to: {
            transform: 'translateX(0)',
            opacity: '1',
          },
        },
      },
    },
  },
  plugins: [],
}

