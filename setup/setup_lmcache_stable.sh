#!/bin/bash

#
# Copyright 2025 Serapheim Dimitropoulos <serapheim.dimitropoulos@weka.io>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# --- Configuration Variables ---
# IMPORTANT: Replace these with your actual details
CONDA_ENV_NAME="amg_stable"
REPO_URL="git@github.com:weka/weka-LMCache.git"
REPO_NAME="LMCache"
COMMIT_HASH="c231e2285ee61a0cbc878d51ed2e7236ac7c0b5d"
BASE_PATH="$HOME/amg-stable"

# VLLM_COMMIT for conditional package installation
VLLM_COMMIT="b6553be1bc75f046b00046a4ad7576364d03c835"

# --- Flags to track actions performed in this script run ---
CONDA_ENV_CREATED=false     # Set to true if the Conda environment was created during this script run
REPO_ACTION_PERFORMED=false # Set to true if the repository was cloned or pulled during this script run

# --- Function to check if a command exists in the system's PATH ---
command_exists () {
    type "$1" &> /dev/null ;
}

# --- Initial Checks: Verify Conda and Git are installed ---
echo "--- Initial Setup Checks ---"
echo "Performing initial checks for required commands (conda, git)..."
if ! command_exists conda; then
    echo "Error: 'conda' command not found."
    echo "Please install Anaconda or Miniconda and ensure it's added to your system's PATH."
    echo "Exiting script."
    exit 1
fi

if ! command_exists git; then
    echo "Error: 'git' command not found."
    echo "Please install Git and ensure it's added to your system's PATH."
    echo "Exiting script."
    exit 1
fi
echo "Conda and Git commands found. Proceeding with setup."

# --- 1. Check and Create Conda Environment ---
echo ""
echo "--- Conda Environment Setup ---"
echo "Checking for Conda environment: '$CONDA_ENV_NAME'..."

# Check if the Conda environment already exists
# `grep -q "\<$CONDA_ENV_NAME\>"` ensures an exact match for the environment name
if ! conda env list | grep -q "\<$CONDA_ENV_NAME\>"; then
    echo "Conda environment '$CONDA_ENV_NAME' not found."
    echo "Creating Conda environment '$CONDA_ENV_NAME' with Python 3.12..."
    # `-y` flag for non-interactive creation (answers yes to prompts)
    conda create -n "$CONDA_ENV_NAME" python=3.12 -y
    if [ $? -ne 0 ]; then # Check the exit status of the previous command
        echo "Error: Failed to create Conda environment '$CONDA_ENV_NAME'."
        echo "Please check your Conda installation, network connection, and permissions."
        echo "Exiting script."
        exit 1
    fi
    CONDA_ENV_CREATED=true
    echo "Conda environment '$CONDA_ENV_NAME' created successfully."
else
    echo "Conda environment '$CONDA_ENV_NAME' already exists."
fi

echo "Activating Conda environment: '$CONDA_ENV_NAME'..."
# Source conda.sh to ensure `conda activate` works correctly in non-interactive scripts
# This line is crucial for Conda commands to be properly initialized in a script
source "$(conda info --base)/etc/profile.d/conda.sh"
conda activate "$CONDA_ENV_NAME"
if [ $? -ne 0 ]; then
    echo "Error: Failed to activate Conda environment '$CONDA_ENV_NAME'."
    echo "Please ensure the environment was created correctly or check your Conda setup."
    echo "Exiting script."
    exit 1
fi
echo "Conda environment '$CONDA_ENV_NAME' activated."

# --- 2. Check and Manage GitHub Repository ---
REPO_PATH="$BASE_PATH/$REPO_NAME"

echo ""
echo "--- GitHub Repository Setup ---"
echo "Checking for repository path: '$REPO_PATH'..."

# Create the base directory if it doesn't exist
if [ ! -d "$BASE_PATH" ]; then
    echo "Base path '$BASE_PATH' not found. Creating it..."
    mkdir -p "$BASE_PATH"
    if [ $? -ne 0 ]; then
        echo "Error: Failed to create base path '$BASE_PATH'."
        echo "Please check your directory permissions. Exiting script."
        exit 1
    fi
fi

# Check if the repository directory exists
if [ ! -d "$REPO_PATH" ]; then
    echo "Repository directory '$REPO_PATH' not found."
    echo "Cloning repository '$REPO_URL' into '$REPO_PATH'..."
    # Using SSH URL for cloning
    git clone "$REPO_URL" "$REPO_PATH"
    if [ $? -ne 0 ]; then
        echo "Error: Failed to clone repository '$REPO_URL'."
        echo "Please ensure you have SSH keys set up for GitHub if using an SSH URL, or check the URL and your network connection. Exiting script."
        exit 1
    fi
    REPO_ACTION_PERFORMED=true
    echo "Repository cloned successfully into '$REPO_PATH'."
elif [ ! -d "$REPO_PATH/.git" ]; then
    # If the directory exists but is not a Git repository
    echo "Warning: Directory '$REPO_PATH' exists but is NOT a Git repository."
    echo "To proceed, please ensure '$REPO_PATH' is either empty or a valid Git repository."
    echo "Exiting script to prevent potential data loss or incorrect operations."
    echo "You may need to manually remove '$REPO_PATH' or initialize it as a Git repository."
    exit 1
else
    # If the directory exists and is a Git repository, pull latest changes
    echo "Repository directory '$REPO_PATH' found and is a Git repository."
    echo "Attempting to pull latest changes from '$REPO_URL'..."
    # `git -C "$REPO_PATH"` runs the git command within the specified directory
    git -C "$REPO_PATH" pull
    if [ $? -ne 0 ]; then
        echo "Error: Failed to pull latest changes from repository in '$REPO_PATH'."
        echo "Please check your network connection or the repository status. Exiting script."
        exit 1
    fi
    REPO_ACTION_PERFORMED=true
    echo "Repository updated (pulled latest changes)."
fi

# --- 3. Check out specific commit ---
echo ""
echo "--- Git Commit Checkout ---"
echo "Checking current commit hash for repository: '$REPO_PATH'..."
# `git rev-parse HEAD` gets the current commit hash
# `2>/dev/null` suppresses any error messages if it's not a git repo (though checked above)
CURRENT_COMMIT=$(git -C "$REPO_PATH" rev-parse HEAD 2>/dev/null)

if [ -z "$CURRENT_COMMIT" ]; then
    echo "Error: Could not determine current commit in '$REPO_PATH'."
    echo "Please ensure it is a valid Git repository. Exiting script."
    exit 1
fi

if [ "$CURRENT_COMMIT" != "$COMMIT_HASH" ]; then
    echo "Current commit ($CURRENT_COMMIT) does not match target commit ($COMMIT_HASH)."
    echo "Checking out commit: $COMMIT_HASH..."
    # `git checkout` switches branches or restores working tree files
    # Use `--force` if you know you want to discard local changes that might conflict
    git -C "$REPO_PATH" checkout "$COMMIT_HASH"
    if [ $? -ne 0 ]; then
        echo "Error: Failed to checkout commit '$COMMIT_HASH'."
        echo "This might be due to uncommitted local changes, an invalid commit hash, or network issues. Exiting script."
        exit 1
    fi
    echo "Successfully checked out commit: $COMMIT_HASH."
else
    echo "Repository is already at the target commit: $COMMIT_HASH."
fi

# --- 4. Install Python packages if Conda environment was just created ---
echo ""
echo "--- Python Package Installation (Conda Environment) ---"
if [ "$CONDA_ENV_CREATED" = true ]; then
    echo "Conda environment '$CONDA_ENV_NAME' was just created."
    echo "Installing initial Python packages into '$CONDA_ENV_NAME'..."
    # Set VLLM_COMMIT for the wheel URL
    export VLLM_COMMIT="$VLLM_COMMIT"
    pip install --no-cache-dir "https://wheels.vllm.ai/${VLLM_COMMIT}/vllm-1.0.0.dev-cp38-abi3-manylinux1_x86_64.whl"
    if [ $? -ne 0 ]; then
        echo "Warning: Failed to install initial Python package (vllm wheel)."
        echo "Please check your internet connection or the wheel URL."
    else
        echo "Initial Python package (vllm wheel) installed successfully."
    fi
    pip install --no-cache-dir py-spy
    if [ $? -ne 0 ]; then
        echo "Warning: Failed to install initial Python package (py-spy)."
        echo "Please check your internet connection or the wheel URL."
    else
        echo "Initial Python package (py-spy) installed successfully."
    fi
    pip install --no-cache-dir scalene
    if [ $? -ne 0 ]; then
        echo "Warning: Failed to install initial Python package (scalene)."
    else
        echo "Initial Python package (scalene) installed successfully."
    fi
    pip install --no-cache-dir pyinstrument
    if [ $? -ne 0 ]; then
        echo "Warning: Failed to install initial Python package (pyinstrument)."
    else
        echo "Initial Python package (pyinstrument) installed successfully."
    fi
    pip install --no-cache-dir line_profiler
    if [ $? -ne 0 ]; then
        echo "Warning: Failed to install initial Python package (line_profiler)."
    else
        echo "Initial Python package (line_profiler) installed successfully."
    fi
else
    echo "Conda environment '$CONDA_ENV_NAME' already existed. Skipping initial package installation."
fi

# --- 5. Install repository dependencies if repo was just pulled/cloned ---
echo ""
echo "--- Python Package Installation (Repository Dependencies) ---"
if [ "$REPO_ACTION_PERFORMED" = true ]; then
    # Check if the required dependency files exist
    if [ -f "$REPO_PATH/requirements/build.txt" ] && \
       [ -f "$REPO_PATH/requirements/common.txt" ] && \
       [ -f "$REPO_PATH/requirements/cuda.txt" ]; then
        echo "Repository was just updated/cloned."
        echo "Installing dependencies from '$REPO_PATH/requirements/build.txt', '$REPO_PATH/requirements/common.txt', and '$REPO_PATH/requirements/cuda.txt'..."
        # Navigate into the repository directory to run the pip command
        (cd "$REPO_PATH" && python3 -m pip install --no-cache-dir -r requirements/build.txt -r requirements/common.txt -r requirements/cuda.txt)
        if [ $? -ne 0 ]; then
            echo "Warning: Failed to install repository dependencies."
            echo "Please check the file paths and content, or your internet connection."
        else
            echo "Repository dependencies installed successfully."
        fi
    else
        echo "One or more required dependency files (requirements/build.txt, common.txt, cuda.txt) not found in the repository at '$REPO_PATH'."
        echo "Skipping repository-specific dependency installation."
    fi
else
    echo "Repository was neither cloned nor pulled in this run. Skipping repository dependency installation."
fi

# --- 6. Install repository in editable mode ---
echo ""
echo "--- Installing Repository in Editable Mode ---"
if [ -d "$REPO_PATH" ]; then
    echo "Navigating to repository root: '$REPO_PATH' and installing in editable mode..."
    (cd "$REPO_PATH" && pip install -e .)
    if [ $? -ne 0 ]; then
        echo "Warning: Failed to install repository in editable mode."
        echo "Please ensure the repository has a setup.py or pyproject.toml file, and check for any installation errors."
    else
        echo "Repository installed in editable mode successfully."
    fi
else
    echo "Repository path '$REPO_PATH' does not exist. Skipping editable mode installation."
fi

# --- 7. Install any one-off's here ---
# Note: Ideally this list should be small and be updated often
#       as we update our packages.
echo ""
echo "--- One-Off: Hot-Patching Transformers Package ---"
if [ -d "$REPO_PATH" ]; then
    echo "Navigating to repository root: '$REPO_PATH' and hot-patching transformers..."
    (cd "$REPO_PATH" && pip install --no-cache-dir "transformers<4.54.0")
    if [ $? -ne 0 ]; then
        echo "Warning: Failed to hot-patch transformers package."
        echo "Please contact the AMG team for troubleshooting."
    else
        echo "Downgraded transformers explicitly - please get rid of this once we update LMCache+vllm."
    fi
else
    echo "Repository path '$REPO_PATH' does not exist. Skipping transformers hot-patch."
fi

echo ""
echo "--- Setup Completed ---"
