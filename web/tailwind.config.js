/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          yellow: '#FAE806',
          gray: '#E5E7EB',
          green: '#6EB42E',
          'green-dark': '#5a9324',
          blue: '#3E4A98',
          'blue-dark': '#2e3a7a',
        },
      },
      fontFamily: {
        sans: ['"Hanken Grotesk"', 'sans-serif'],
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
