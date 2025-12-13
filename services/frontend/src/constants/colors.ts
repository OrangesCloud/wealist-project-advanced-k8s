/**
 * Predefined color palette for custom fields (Stages, Roles, Importance)
 * Each color has a name, hex value, and Tailwind CSS class
 */

export interface ColorOption {
  name: string;
  hex: string;
  bgClass: string; // Tailwind background class
  textClass: string; // Tailwind text class for labels
}

export const CUSTOM_FIELD_COLORS: ColorOption[] = [
  {
    name: '파란색',
    hex: '#3B82F6',
    bgClass: 'bg-blue-500',
    textClass: 'text-blue-500',
  },
  {
    name: '하늘색',
    hex: '#06B6D4',
    bgClass: 'bg-cyan-500',
    textClass: 'text-cyan-500',
  },
  {
    name: '청록색',
    hex: '#14B8A6',
    bgClass: 'bg-teal-500',
    textClass: 'text-teal-500',
  },
  {
    name: '초록색',
    hex: '#22C55E',
    bgClass: 'bg-green-500',
    textClass: 'text-green-500',
  },
  {
    name: '연두색',
    hex: '#84CC16',
    bgClass: 'bg-lime-500',
    textClass: 'text-lime-500',
  },
  {
    name: '노란색',
    hex: '#EAB308',
    bgClass: 'bg-yellow-500',
    textClass: 'text-yellow-500',
  },
  {
    name: '주황색',
    hex: '#F97316',
    bgClass: 'bg-orange-500',
    textClass: 'text-orange-500',
  },
  {
    name: '빨간색',
    hex: '#EF4444',
    bgClass: 'bg-red-500',
    textClass: 'text-red-500',
  },
  {
    name: '분홍색',
    hex: '#EC4899',
    bgClass: 'bg-pink-500',
    textClass: 'text-pink-500',
  },
  {
    name: '보라색',
    hex: '#A855F7',
    bgClass: 'bg-purple-500',
    textClass: 'text-purple-500',
  },
  {
    name: '남색',
    hex: '#6366F1',
    bgClass: 'bg-indigo-500',
    textClass: 'text-indigo-500',
  },
  {
    name: '회색',
    hex: '#6B7280',
    bgClass: 'bg-gray-500',
    textClass: 'text-gray-500',
  },
];

/**
 * Get color option by hex value
 */
export const getColorByHex = (hex: string): ColorOption | undefined => {
  return CUSTOM_FIELD_COLORS.find((color) => color.hex.toLowerCase() === hex.toLowerCase());
};

/**
 * Get default color by index (for auto-assignment)
 */
export const getDefaultColorByIndex = (index: number): ColorOption => {
  return CUSTOM_FIELD_COLORS[index % CUSTOM_FIELD_COLORS.length];
};
