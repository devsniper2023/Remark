import { Comment, Node } from '@app/common/types';

/**
 * Filters tree node
 */
export function filterTree(tree: Node[], fn: (node: Node) => boolean): Node[] {
  let filtered = false;
  const newTree = tree.reduce<Node[]>((tree, node) => {
    if (!fn(node)) {
      filtered = true;
      return tree;
    }
    const newNode: Node = !node.replies ? node : { ...node, replies: filterTree(node.replies, fn) };
    if (newNode !== node) {
      filtered = true;
    }
    tree.push(newNode);
    return tree;
  }, []);
  if (!filtered) return tree;
  return newTree;
}

export function findPinnedComments(thread: Node): Comment[] {
  let result: Comment[] = [];

  if (thread.comment.pin) {
    result = result.concat(thread.comment);
  }

  if (thread.replies) {
    result = result.concat(
      thread.replies.reduce((acc: Comment[], thread: Node) => acc.concat(findPinnedComments(thread)), [])
    );
  }

  return result;
}

export function getPinnedComments(threads: Node[]): Comment[] {
  return threads.reduce((acc: Comment[], thread: Node) => acc.concat(findPinnedComments(thread)), []);
}
