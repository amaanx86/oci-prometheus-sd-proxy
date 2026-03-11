# Sphinx configuration file for oci-prometheus-sd-proxy documentation

project = "oci-prometheus-sd-proxy"
copyright = "2026, Amaan Ul Haq Siddiqui"
author = "Amaan Ul Haq Siddiqui"
version = "1.1"
release = "1.1.0"

extensions = [
    "myst_parser",  # Markdown support
]

# Markdown file extensions
source_suffix = {
    ".rst": None,
    ".md": "myst-nb",
}

# Theme
html_theme = "furo"

html_static_path = ["_static"]

# Exclude build files
exclude_patterns = ["_build", "Thumbs.db", ".DS_Store"]

# MyST parser options
myst_enable_extensions = [
    "colon_fence",
    "html_image",
]
