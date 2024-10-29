import colorsys

def generate_pastel_colors(num_colors):
    """
    Generates a list of pastel hex colors evenly spaced around the color wheel.

    Parameters:
    num_colors (int): The number of distinct pastel colors to generate.

    Returns:
    list: A list of hex color codes.
    """
    pastel_colors = []

    # Evenly space hues around the color wheel
    for i in range(num_colors):
        hue = i / num_colors  # Evenly spaced hues
        # Use high value and low-to-medium saturation to get pastel-like colors
        rgb = colorsys.hsv_to_rgb(hue, 0.4, 1.0)  # S = 0.4 for pastel, V = 1.0 for brightness
        # Convert to hex
        pastel_colors.append('#{:02x}{:02x}{:02x}'.format(int(rgb[0] * 255), int(rgb[1] * 255), int(rgb[2] * 255)))

    return pastel_colors

def assign_colors_to_ids(ids):
    """
    Assigns a unique pastel color to each ID.

    Parameters:
    ids (list): A list of unique IDs.

    Returns:
    dict: A dictionary mapping each ID to a unique pastel color.
    """
    num_ids = len(ids)
    pastel_colors = generate_pastel_colors(num_ids)
    return {id_: color for id_, color in zip(ids, pastel_colors)}
