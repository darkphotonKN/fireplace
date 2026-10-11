import FireIcon from './icons/FireIcon';

// Sized in em so the mark scales with whatever heading it sits beside: place
// it inside (or beside, sharing the font size of) the wordmark's text. The
// fire fills ~80% of its box (the rest is room for its shadow), so 1.15em
// keeps it a touch under the height of the lettering.
const Logo = () => {
  return <FireIcon width="1.15em" height="1.15em" className="shrink-0" />;
};

export default Logo;
