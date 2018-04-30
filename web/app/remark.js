import 'babel-polyfill'; // TODO: remove it
import 'common/polyfills'; // TODO: check it

import { h, render } from 'preact';
import Root from './components/root';
import ListComments from './components/list-comments'; // TODO: temp solution for extracting styles

import { NODE_ID } from './common/constants';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init() {
  const node = document.getElementById(NODE_ID);

  if (!node) {
    console.error('Remark42: Can\'t find root node.');
    return;
  }

  render(<Root/>, node.parentElement, node);
}
