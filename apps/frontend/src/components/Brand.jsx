import { Link } from "react-router-dom";

const LOGO_SIZES = {
  compact: 36,
  hero: 72,
  landing: 112,
};

/**
 * Product mark: raven favicon + Huginn wordmark.
 * @param {"landing" | "hero" | "compact"} [size]
 * @param {string} [subtitle]
 * @param {boolean} [linkHome]
 * @param {boolean} [showWordmark]
 */
export default function Brand({ size = "compact", subtitle, linkHome = false, showWordmark = true }) {
  const px = LOGO_SIZES[size] || LOGO_SIZES.compact;
  const content = (
    <div className={`brand-mark brand-mark--${size}`}>
      <img src="/favicon.ico" alt="" className="brand-logo" width={px} height={px} />
      {showWordmark ? (
        <div className="brand-text">
          <span className="brand-wordmark">Huginn</span>
          {subtitle ? <span className="brand-subtitle">{subtitle}</span> : null}
        </div>
      ) : null}
    </div>
  );

  if (linkHome) {
    return (
      <Link to="/sessions" className="brand-link" aria-label="Huginn home">
        {content}
      </Link>
    );
  }

  return content;
}
