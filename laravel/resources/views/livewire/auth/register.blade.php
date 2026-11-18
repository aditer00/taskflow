<div class="max-w-[360px] mx-auto pt-6 sm:pt-12 pb-10">
    <div class="text-center mb-8">
        <div class="w-10 h-10 mx-auto mb-5 rounded-[8px] bg-[#0a1530] text-[#ffffff] font-mono text-base font-semibold inline-flex items-center justify-center" aria-hidden="true">
            T
        </div>
        <h1 class="text-[22px] font-semibold tracking-[-0.02em] leading-[1.3] text-[#1a1a1a]">
            Your task workspace.
        </h1>
        <p class="text-[22px] font-semibold tracking-[-0.02em] leading-[1.3] text-[#787671]">
            Create your TaskFlow account
        </p>
    </div>

    <form wire:submit="register" class="space-y-4">
        <div>
            <label for="reg-name" class="block text-xs font-medium text-[#5d5b54] mb-1.5">Name</label>
            <input id="reg-name" type="text" wire:model="name" required autocomplete="name"
                placeholder="Enter your name..."
                class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3.5 py-2 text-sm text-[#1a1a1a] placeholder:text-[#a4a097] focus:border-[#0075de]">
            @error('name')
                <p class="text-[#e03131] text-xs font-medium mt-1.5" role="alert">{{ $message }}</p>
            @enderror
        </div>

        <div>
            <label for="reg-email" class="block text-xs font-medium text-[#5d5b54] mb-1.5">Email</label>
            <input id="reg-email" type="email" wire:model="email" required autocomplete="email"
                placeholder="Enter your email address..."
                class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3.5 py-2 text-sm text-[#1a1a1a] placeholder:text-[#a4a097] focus:border-[#0075de]">
            @error('email')
                <p class="text-[#e03131] text-xs font-medium mt-1.5" role="alert">{{ $message }}</p>
            @enderror
        </div>

        <div>
            <label for="reg-password" class="block text-xs font-medium text-[#5d5b54] mb-1.5">Password</label>
            <input id="reg-password" type="password" wire:model="password" required autocomplete="new-password"
                placeholder="At least 8 characters..."
                class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3.5 py-2 text-sm text-[#1a1a1a] placeholder:text-[#a4a097] focus:border-[#0075de]">
            @error('password')
                <p class="text-[#e03131] text-xs font-medium mt-1.5" role="alert">{{ $message }}</p>
            @enderror
        </div>

        <div>
            <label for="reg-password-confirm" class="block text-xs font-medium text-[#5d5b54] mb-1.5">Confirm password</label>
            <input id="reg-password-confirm" type="password" wire:model="password_confirmation" required autocomplete="new-password"
                placeholder="Repeat your password..."
                class="w-full min-h-[44px] bg-[#ffffff] border border-[#c8c4be] rounded-[8px] px-3.5 py-2 text-sm text-[#1a1a1a] placeholder:text-[#a4a097] focus:border-[#0075de]">
        </div>

        <button type="submit" wire:loading.attr="disabled"
            class="w-full min-h-[44px] px-4 py-2.5 bg-[#0075de] hover:bg-[#005bab] disabled:opacity-60 text-[#ffffff] text-sm font-medium rounded-[8px] cursor-pointer">
            <span wire:loading.remove wire:target="register">Continue</span>
            <span wire:loading wire:target="register">Creating account...</span>
        </button>
    </form>

    <p class="mt-8 text-center text-[13px] text-[#5d5b54]">
        Already have an account?
        <a href="{{ route('login') }}" class="inline-flex items-center min-h-[44px] px-1 font-medium text-[#5d5b54] underline underline-offset-4 hover:text-[#1a1a1a]">
            Sign in
        </a>
    </p>
</div>
