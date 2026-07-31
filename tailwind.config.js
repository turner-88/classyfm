/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./web/static/js/**/*.js",
  ],
  // Progress-bar widths (today-programs.html, schedule.js) are set by toggling
  // one of these classes instead of an inline `style="width:...` attribute, so
  // they render under a CSP with no `style-src 'unsafe-inline'`. Progress is
  // always an integer 0-100 (see models.Progress), so 0%-100% covers every value.
  // `animate-marquee` is safelisted because Tailwind only emits an animation's
  // @keyframes alongside the utility class itself, and reaching it through
  // `@apply` from a component (.marquee in tailwind.css) doesn't count - the
  // animation: declaration lands in the output with nothing to reference. Nothing
  // in the markup ever uses the bare class; this entry exists to get the keyframes
  // into the build.
  safelist: ["animate-marquee"].concat(
    Array.from({ length: 101 }, (_, i) => `w-[${i}%]`),
  ),
  theme: {
    extend: {
      colors: {
        // Sampled from the logo artwork (web/static/img/logo.png): navy #292E62
        // for the broadcast arcs, red #E32229 for the transmitter dot.
        brand: {
          50: "#f3f4f9",
          100: "#e4e6f0",
          200: "#c6cade",
          300: "#9da3c4",
          400: "#6b73a2",
          DEFAULT: "#292e62",
          600: "#232760",
          700: "#1b1f4a",
          800: "#14173a",
          900: "#0d0f26",
          // Aliases kept so existing `from-brand to-brand-dark` gradients stay valid.
          dark: "#1b1f4a",
          light: "#6b73a2",
        },
        // Reserved for live/on-air state only — it is the studio on-air lamp,
        // not a general accent. Also doubles as the focus-ring colour.
        signal: {
          DEFAULT: "#e32229",
          dark: "#b81a20",
          light: "#fdecec",
        },
        // Neutral scale tinted toward the brand navy, replacing Tailwind's stock
        // gray so every existing text-gray-*/border-gray-* utility picks it up.
        gray: {
          50: "#f6f7fa",
          100: "#edeef4",
          200: "#dee0ea",
          300: "#c3c7d7",
          400: "#9096ac",
          500: "#6b7189",
          600: "#4e5468",
          700: "#3a3f50",
          800: "#262a38",
          900: "#171a24",
        },
        ink: "#171a24",
      },
      // Deliberately a two-typeface system: Outfit for headings, Plus Jakarta
      // Sans for body. No `serif` override — long-form prose (.prose-article)
      // reads in the body sans too. Don't add a third family.
      fontFamily: {
        sans: ["Plus Jakarta Sans", "system-ui", "sans-serif"],
        heading: ["Outfit", "system-ui", "sans-serif"],
      },
      maxWidth: {
        prose: "68ch",
      },
      boxShadow: {
        card: "0 1px 2px 0 rgb(23 26 36 / 0.05)",
        lift: "0 12px 28px -12px rgb(41 46 98 / 0.35)",
        // For the floating radio player. `lift` is offset down with a -12px
        // spread, so it casts below the element and nothing wraps its sides or
        // top — fine for a card sitting in a page, too weak for something
        // hovering over arbitrary content. These layers go the other way: a
        // hairline ring plus two un-offset halos define the edge all the way
        // around, and only the last layer adds downward depth.
        glass: [
          "0 0 0 1px rgb(41 46 98 / 0.06)",
          "0 2px 6px rgb(41 46 98 / 0.08)",
          "0 8px 20px -4px rgb(41 46 98 / 0.18)",
          "0 24px 48px -16px rgb(41 46 98 / 0.32)",
        ].join(", "),
      },
      keyframes: {
        // The live dot breathing — softer than Tailwind's animate-pulse.
        onair: {
          "0%, 100%": { opacity: "1", transform: "scale(1)" },
          "50%": { opacity: "0.4", transform: "scale(0.82)" },
        },
        // An arc expanding away from the play button, echoing the logo mark.
        ripple: {
          "0%": { opacity: "0.5", transform: "scale(0.85)" },
          "100%": { opacity: "0", transform: "scale(2.2)" },
        },
        eq: {
          "0%, 100%": { transform: "scaleY(0.3)" },
          "50%": { transform: "scaleY(1)" },
        },
        "fade-up": {
          "0%": { opacity: "0", transform: "translateY(10px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        // A line too wide for the floating player slides left far enough to
        // reveal its tail, holds so it can be read, then returns. The distance
        // is per-element, so it arrives as a custom property marquee.js sets.
        // Each travel leg gets 36% of the cycle and each hold 14%, which lets
        // JS control the whole thing with one duration.
        marquee: {
          "0%, 14%": { transform: "translateX(0)" },
          "50%, 64%": { transform: "translateX(var(--marquee-shift, 0px))" },
          "100%": { transform: "translateX(0)" },
        },
      },
      animation: {
        onair: "onair 2s ease-in-out infinite",
        ripple: "ripple 2.4s ease-out infinite",
        eq: "eq 900ms ease-in-out infinite",
        "fade-up": "fade-up 500ms cubic-bezier(0.22, 1, 0.36, 1) both",
        marquee: "marquee var(--marquee-duration, 8s) ease-in-out infinite",
      },
    },
  },
  plugins: [],
};
