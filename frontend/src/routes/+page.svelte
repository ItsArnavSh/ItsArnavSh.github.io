<script>
	import { onMount } from 'svelte';
	import Hero from './Hero.svelte';

	let isScrolled = false;

	onMount(() => {
		window.addEventListener('scroll', () => {
			isScrolled = window.scrollY > 20;
		});
	});
</script>

<main class="bg-[#faf8f6]">
	<section id="hero" class="min-h-screen pt-20">
		<Hero />
	</section>

	<!-- Other sections remain the same -->
</main>

<style>
	:global(body) {
		background-color: #faf8f6;
	}

	:global(.bg-coral-400) {
		background-color: #ff906d;
	}

	:global(.bg-coral-500) {
		background-color: #ff7f5d;
	}
</style>
