#!/usr/bin/env python3
"""Generate PWA icons from logo.svg"""
import os
from pathlib import Path

try:
    from PIL import Image, ImageDraw
except ImportError:
    print("Error: Pillow is required. Install with: pip install Pillow")
    exit(1)

def create_icon(size: int, svg_path: str, output_path: str):
    """Create a PNG icon from an SVG or create a solid color placeholder"""

    # Create a simple placeholder icon with TeamWERK branding
    img = Image.new('RGBA', (size, size), color=(0, 0, 0, 255))
    draw = ImageDraw.Draw(img)

    # Draw yellow square in center
    margin = int(size * 0.1)
    draw.rectangle(
        [(margin, margin), (size - margin, size - margin)],
        fill=(250, 232, 6, 255),  # #FAE806 (TeamWERK yellow)
    )

    # Draw "T" text (TeamWERK initial)
    if size >= 192:
        # For larger icons, draw a simple "T"
        line_width = max(1, int(size * 0.15))
        text_margin = int(size * 0.25)
        # Vertical line
        draw.rectangle(
            [(size//2 - line_width//2, text_margin), (size//2 + line_width//2, size - text_margin)],
            fill=(0, 0, 0, 255)
        )
        # Horizontal line
        draw.rectangle(
            [(text_margin, text_margin), (size - text_margin, text_margin + line_width)],
            fill=(0, 0, 0, 255)
        )

    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    img.save(output_path, 'PNG')
    print(f"✓ Generated {output_path} ({size}x{size})")

if __name__ == '__main__':
    base_dir = Path(__file__).parent
    icons_dir = base_dir / 'public' / 'icons'

    print("Generating PWA icons...")
    create_icon(192, str(base_dir / 'public' / 'logo.svg'), str(icons_dir / 'icon-192.png'))
    create_icon(512, str(base_dir / 'public' / 'logo.svg'), str(icons_dir / 'icon-512.png'))
    print("\nDone! Icons are ready for PWA.")
