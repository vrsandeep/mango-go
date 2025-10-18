import { checkAuth } from './auth.js';

document.addEventListener('DOMContentLoaded', async () => {
  const currentUser = await checkAuth();
  if (!currentUser) return;

  const tagsList = document.getElementById('tags-list');

  const loadTags = async () => {
    const response = await fetch('/api/tags');
    const tags = await response.json();

    tagsList.innerHTML = '';
    if (tags && tags.length > 0) {
      tags.forEach(tag => {
        const li = document.createElement('li');
        li.className = 'tag-item';
        const tagCount = tag.folder_count || 0;
        li.innerHTML = `<a href="/tags/${tag.id}">${tag.name} <span class="tag-count">(${tagCount})</span></a>`;
        tagsList.appendChild(li);
      });
    } else {
      tagsList.innerHTML =
        '<li>No tags found. Add tags to your series from the chapters page.</li>';
    }
  };

  loadTags();
});
