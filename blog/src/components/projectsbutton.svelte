<script lang="ts">
	import '../routes/layout.css';
</script>

<div class="relative flex flex-col gap-2">
	<!-- Small heading -->
	<div
		class="text-(--lightbg) bayon text-3xl drop-shadow-[4px_4px_0px_rgba(0,0,0,1)]
                 translate-y-2"
	>
		WANNA SEE SOME
	</div>

	<!-- Big heading -->
	<div>
		<p class="bayon text-8xl text-(--lightbg) leading-none">
			COOL <a
				href="/projects"
				class="text-(--accent) drop-shadow-[8px_6px_0px_rgba(0,0,0,1)] hover:drop-shadow-[12px_12px_0px_rgba(0,0,0,1)]
                         transition-all duration-300"
			>
				Projects
			</a>?
		</p>
	</div>
</div>
