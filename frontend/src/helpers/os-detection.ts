/**
 * OS detection utilities for determining the user's operating system
 */

/**
 * Detects the current operating system based on the user agent string
 * @returns 'windows' for Windows, 'unix' for Unix-like systems (macOS, Linux)
 */
export function detectOperatingSystem(): 'windows' | 'unix' {
  const userAgent = navigator.userAgent.toLowerCase();
  
  // Check for Windows
  if (userAgent.includes('win')) {
    return 'windows';
  }
  
  // Default to unix for macOS, Linux, and other Unix-like systems
  return 'unix';
}