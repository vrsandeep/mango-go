/**
 * Path utilities for folder path management
 * Common functions used across plugins and subscription manager
 */

// Global variable to store the library path
let libraryPath = '';

/**
 * Loads the library path from the server configuration
 */
const loadLibraryPath = async () => {
  try {
    const response = await fetch('/api/config');
    const config = await response.json();
    libraryPath = config.library_path || '';

    // Normalize the library path - remove leading ./ and ensure it ends with /
    if (libraryPath.startsWith('./')) {
      libraryPath = libraryPath.substring(2);
    }
    if (libraryPath && !libraryPath.endsWith('/')) {
      libraryPath += '/';
    }
  } catch (error) {
    console.error('Failed to load library path:', error);
    libraryPath = '';
  }
};

/**
 * Sanitizes a folder path by removing invalid characters and normalizing slashes
 * @param {string} path - The path to sanitize
 * @returns {string|null} - The sanitized path or null if invalid
 */
const sanitizePath = path => {
  if (!path) return null;

  // Remove leading/trailing whitespace and slashes
  let sanitized = path.trim().replace(/^\/+|\/+$/g, '');

  if (!sanitized) return null;

  // Replace multiple slashes with single slash
  sanitized = sanitized.replace(/\/+/g, '/');

  // Remove invalid characters for filesystem
  sanitized = sanitized.replace(/[<>:"|?*\x00-\x1f]/g, '');

  // Remove leading dots (security)
  sanitized = sanitized.replace(/^\.+/, '');

  if (!sanitized) return null;

  return sanitized;
};

/**
 * Generates the default folder path (library path + manga name)
 * @param {string} mangaName - The manga/series name
 * @returns {string} - The default folder path
 */
const getDefaultFolderPath = mangaName => {
  if (!mangaName) return libraryPath;
  return libraryPath + mangaName;
};

/**
 * Pre-fills a custom path input with the appropriate value
 * @param {HTMLInputElement} inputElement - The input element to pre-fill
 * @param {string} existingPath - The existing folder path (if any)
 * @param {string} mangaName - The manga/series name for default path generation
 */
const prefillCustomPath = (inputElement, existingPath = null, mangaName = '') => {
  if (inputElement) {
    if (existingPath) {
      // If there's an existing custom path, use it
      inputElement.value = existingPath;
    } else {
      // If no existing path, use default (library path + manga name)
      inputElement.value = getDefaultFolderPath(mangaName);
    }
  }
};

/**
 * Gets the current library path
 * @returns {string} - The current library path
 */
const getLibraryPath = () => {
  return libraryPath;
};

// Export functions for use in other modules
window.PathUtils = {
  loadLibraryPath,
  sanitizePath,
  prefillCustomPath,
  getLibraryPath,
  getDefaultFolderPath,
};
