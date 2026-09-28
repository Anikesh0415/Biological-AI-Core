/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'bio-glow': '#00f0ff',
        'cosmic-glow': '#a200ff',
      }
    },
  },
  plugins: [],
}
